package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
)

// SnapshotRepository implementa repository.SnapshotRepository sobre SQLite.
// Snapshot, sus AccountBalance y sus ExchangeRate se escriben siempre juntos
// dentro de una misma transacción: un snapshot sin desglose completo no
// tiene sentido de negocio (RF-05, RF-06).
type SnapshotRepository struct {
	db *sql.DB
}

func NewSnapshotRepository(db *sql.DB) *SnapshotRepository {
	return &SnapshotRepository{db: db}
}

var _ repository.SnapshotRepository = (*SnapshotRepository)(nil)

const (
	snapshotColumns = "id, snapshot_date, notes, created_at, updated_at"
	rateColumns     = "id, snapshot_id, currency_code, rate_to_ars, source, fetched_at"
	balanceColumns  = "id, snapshot_id, account_id, balance_original, balance_ars, percentage_of_total"
)

func (r *SnapshotRepository) Create(ctx context.Context, snapshot *domain.Snapshot, balances []domain.AccountBalance, rates []domain.ExchangeRate) error {
	if snapshot.ID == "" {
		snapshot.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	snapshot.CreatedAt = now
	snapshot.UpdatedAt = now

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op si ya se hizo Commit

	if _, err := tx.ExecContext(ctx, `
		INSERT INTO snapshots (`+snapshotColumns+`)
		VALUES (?, ?, ?, ?, ?)
	`,
		snapshot.ID, snapshot.SnapshotDate.UTC().Format(time.RFC3339Nano), snapshot.Notes,
		snapshot.CreatedAt.Format(time.RFC3339Nano), snapshot.UpdatedAt.Format(time.RFC3339Nano),
	); err != nil {
		if isUniqueConstraintErr(err) {
			return repository.ErrConflict
		}
		return fmt.Errorf("insertando snapshot: %w", err)
	}

	if err := insertRates(ctx, tx, snapshot.ID, rates); err != nil {
		return err
	}
	if err := insertBalances(ctx, tx, snapshot.ID, balances); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *SnapshotRepository) Update(ctx context.Context, snapshot *domain.Snapshot, balances []domain.AccountBalance, rates []domain.ExchangeRate) error {
	snapshot.UpdatedAt = time.Now().UTC()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback() //nolint:errcheck // no-op si ya se hizo Commit

	res, err := tx.ExecContext(ctx, `
		UPDATE snapshots SET snapshot_date = ?, notes = ?, updated_at = ? WHERE id = ?
	`, snapshot.SnapshotDate.UTC().Format(time.RFC3339Nano), snapshot.Notes, snapshot.UpdatedAt.Format(time.RFC3339Nano), snapshot.ID)
	if err != nil {
		if isUniqueConstraintErr(err) {
			return repository.ErrConflict
		}
		return fmt.Errorf("actualizando snapshot %s: %w", snapshot.ID, err)
	}
	if err := checkRowsAffected(res); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `DELETE FROM exchange_rates WHERE snapshot_id = ?`, snapshot.ID); err != nil {
		return fmt.Errorf("limpiando cotizaciones del snapshot %s: %w", snapshot.ID, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM account_balances WHERE snapshot_id = ?`, snapshot.ID); err != nil {
		return fmt.Errorf("limpiando saldos del snapshot %s: %w", snapshot.ID, err)
	}

	if err := insertRates(ctx, tx, snapshot.ID, rates); err != nil {
		return err
	}
	if err := insertBalances(ctx, tx, snapshot.ID, balances); err != nil {
		return err
	}

	return tx.Commit()
}

// Delete borra el snapshot; sus account_balances y exchange_rates se van en
// cascada (ON DELETE CASCADE, ver migración 0001).
func (r *SnapshotRepository) Delete(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM snapshots WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("eliminando snapshot %s: %w", id, err)
	}
	return checkRowsAffected(res)
}

func (r *SnapshotRepository) GetByID(ctx context.Context, id string) (*domain.Snapshot, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+snapshotColumns+` FROM snapshots WHERE id = ?`, id)
	snap, err := scanSnapshot(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("buscando snapshot %s: %w", id, err)
	}
	return snap, nil
}

// ExistsForMonth verifica si ya existe un snapshot cuya snapshot_date caiga
// en el mismo mes que date (RF-08). La comparación es lexicográfica sobre el
// formato RFC3339Nano en UTC, que preserva el orden cronológico.
func (r *SnapshotRepository) ExistsForMonth(ctx context.Context, date time.Time) (bool, error) {
	date = date.UTC()
	monthStart := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	var exists int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1 FROM snapshots WHERE snapshot_date >= ? AND snapshot_date < ? LIMIT 1
	`, monthStart.Format(time.RFC3339Nano), monthEnd.Format(time.RFC3339Nano)).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("verificando snapshot del mes de %s: %w", date.Format("2006-01"), err)
	}
	return true, nil
}

func (r *SnapshotRepository) List(ctx context.Context, filter repository.SnapshotFilter) ([]domain.Snapshot, error) {
	query := "SELECT " + snapshotColumns + " FROM snapshots WHERE 1 = 1"
	var args []interface{}

	if filter.From != nil {
		query += " AND snapshot_date >= ?"
		args = append(args, filter.From.UTC().Format(time.RFC3339Nano))
	}
	if filter.To != nil {
		query += " AND snapshot_date <= ?"
		args = append(args, filter.To.UTC().Format(time.RFC3339Nano))
	}
	query += " ORDER BY snapshot_date ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listando snapshots: %w", err)
	}
	defer rows.Close()

	snapshots := []domain.Snapshot{}
	for rows.Next() {
		snap, err := scanSnapshot(rows)
		if err != nil {
			return nil, fmt.Errorf("leyendo snapshot: %w", err)
		}
		snapshots = append(snapshots, *snap)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return snapshots, nil
}

func (r *SnapshotRepository) GetBalances(ctx context.Context, snapshotID string) ([]domain.AccountBalance, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+balanceColumns+` FROM account_balances WHERE snapshot_id = ? ORDER BY account_id ASC`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("listando saldos del snapshot %s: %w", snapshotID, err)
	}
	defer rows.Close()

	balances := []domain.AccountBalance{}
	for rows.Next() {
		b, err := scanBalance(rows)
		if err != nil {
			return nil, fmt.Errorf("leyendo saldo: %w", err)
		}
		balances = append(balances, *b)
	}
	return balances, rows.Err()
}

func (r *SnapshotRepository) GetRates(ctx context.Context, snapshotID string) ([]domain.ExchangeRate, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+rateColumns+` FROM exchange_rates WHERE snapshot_id = ? ORDER BY currency_code ASC`, snapshotID)
	if err != nil {
		return nil, fmt.Errorf("listando cotizaciones del snapshot %s: %w", snapshotID, err)
	}
	defer rows.Close()

	rates := []domain.ExchangeRate{}
	for rows.Next() {
		rt, err := scanRate(rows)
		if err != nil {
			return nil, fmt.Errorf("leyendo cotización: %w", err)
		}
		rates = append(rates, *rt)
	}
	return rates, rows.Err()
}

// BalanceHistory devuelve la serie temporal de saldos de una cuenta entre
// dos fechas (RF-23), ordenada cronológicamente por la fecha del snapshot al
// que pertenece cada saldo.
func (r *SnapshotRepository) BalanceHistory(ctx context.Context, accountID string, from, to time.Time) ([]domain.AccountBalance, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT ab.id, ab.snapshot_id, ab.account_id, ab.balance_original, ab.balance_ars, ab.percentage_of_total
		FROM account_balances ab
		JOIN snapshots s ON s.id = ab.snapshot_id
		WHERE ab.account_id = ? AND s.snapshot_date >= ? AND s.snapshot_date <= ?
		ORDER BY s.snapshot_date ASC
	`, accountID, from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, fmt.Errorf("buscando historial de saldos de la cuenta %s: %w", accountID, err)
	}
	defer rows.Close()

	balances := []domain.AccountBalance{}
	for rows.Next() {
		b, err := scanBalance(rows)
		if err != nil {
			return nil, fmt.Errorf("leyendo saldo histórico: %w", err)
		}
		balances = append(balances, *b)
	}
	return balances, rows.Err()
}

func insertRates(ctx context.Context, tx *sql.Tx, snapshotID string, rates []domain.ExchangeRate) error {
	for i := range rates {
		if rates[i].ID == "" {
			rates[i].ID = uuid.NewString()
		}
		rates[i].SnapshotID = snapshotID
		if rates[i].FetchedAt.IsZero() {
			rates[i].FetchedAt = time.Now().UTC()
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO exchange_rates (`+rateColumns+`)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			rates[i].ID, rates[i].SnapshotID, string(rates[i].CurrencyCode), rates[i].RateToARS.String(),
			rates[i].Source, rates[i].FetchedAt.Format(time.RFC3339Nano),
		); err != nil {
			return fmt.Errorf("insertando cotización %s del snapshot %s: %w", rates[i].CurrencyCode, snapshotID, err)
		}
	}
	return nil
}

func insertBalances(ctx context.Context, tx *sql.Tx, snapshotID string, balances []domain.AccountBalance) error {
	for i := range balances {
		if balances[i].ID == "" {
			balances[i].ID = uuid.NewString()
		}
		balances[i].SnapshotID = snapshotID
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO account_balances (`+balanceColumns+`)
			VALUES (?, ?, ?, ?, ?, ?)
		`,
			balances[i].ID, balances[i].SnapshotID, balances[i].AccountID,
			balances[i].BalanceOriginal.String(), balances[i].BalanceARS.String(), balances[i].PercentageOfTotal.String(),
		); err != nil {
			return fmt.Errorf("insertando saldo de la cuenta %s en el snapshot %s: %w", balances[i].AccountID, snapshotID, err)
		}
	}
	return nil
}

func scanSnapshot(s rowScanner) (*domain.Snapshot, error) {
	var (
		id, snapshotDate, createdAt, updatedAt string
		notes                                  sql.NullString
	)
	if err := s.Scan(&id, &snapshotDate, &notes, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	snap := &domain.Snapshot{ID: id}
	if notes.Valid {
		v := notes.String
		snap.Notes = &v
	}
	if t, err := time.Parse(time.RFC3339Nano, snapshotDate); err == nil {
		snap.SnapshotDate = t
	}
	if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		snap.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		snap.UpdatedAt = t
	}
	return snap, nil
}

func scanBalance(s rowScanner) (*domain.AccountBalance, error) {
	var (
		id, snapshotID, accountID                      string
		balanceOriginal, balanceARS, percentageOfTotal string
	)
	if err := s.Scan(&id, &snapshotID, &accountID, &balanceOriginal, &balanceARS, &percentageOfTotal); err != nil {
		return nil, err
	}

	b := &domain.AccountBalance{ID: id, SnapshotID: snapshotID, AccountID: accountID}
	b.BalanceOriginal, _ = decimal.NewFromString(balanceOriginal)
	b.BalanceARS, _ = decimal.NewFromString(balanceARS)
	b.PercentageOfTotal, _ = decimal.NewFromString(percentageOfTotal)
	return b, nil
}

func scanRate(s rowScanner) (*domain.ExchangeRate, error) {
	var (
		id, snapshotID, currencyCode, rateToARS, source, fetchedAt string
	)
	if err := s.Scan(&id, &snapshotID, &currencyCode, &rateToARS, &source, &fetchedAt); err != nil {
		return nil, err
	}

	r := &domain.ExchangeRate{ID: id, SnapshotID: snapshotID, CurrencyCode: domain.Currency(currencyCode), Source: source}
	r.RateToARS, _ = decimal.NewFromString(rateToARS)
	if t, err := time.Parse(time.RFC3339Nano, fetchedAt); err == nil {
		r.FetchedAt = t
	}
	return r, nil
}
