package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
)

// AccountRepository implementa repository.AccountRepository sobre SQLite,
// usando database/sql directamente (sin query builder) para no sumar
// dependencias adicionales.
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository construye el repositorio de cuentas.
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// compile-time check: AccountRepository debe cumplir la interfaz del dominio.
var _ repository.AccountRepository = (*AccountRepository)(nil)

const accountColumns = "id, name, type, currency, provider_group, is_invested, is_active, created_at, updated_at"

func (r *AccountRepository) Create(ctx context.Context, account *domain.Account) error {
	if account.ID == "" {
		account.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	account.CreatedAt = now
	account.UpdatedAt = now
	if !account.IsActive {
		account.IsActive = true // por defecto una cuenta nueva está activa (RF-01)
	}

	_, err := r.db.ExecContext(ctx, `
		INSERT INTO accounts (`+accountColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`,
		account.ID, account.Name, string(account.Type), string(account.Currency), account.ProviderGroup,
		account.IsInvested, account.IsActive,
		account.CreatedAt.Format(time.RFC3339Nano), account.UpdatedAt.Format(time.RFC3339Nano),
	)
	if err != nil {
		return fmt.Errorf("insertando cuenta: %w", err)
	}
	return nil
}

func (r *AccountRepository) Update(ctx context.Context, account *domain.Account) error {
	account.UpdatedAt = time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE accounts
		SET name = ?, type = ?, currency = ?, provider_group = ?, is_invested = ?, is_active = ?, updated_at = ?
		WHERE id = ?
	`,
		account.Name, string(account.Type), string(account.Currency), account.ProviderGroup,
		account.IsInvested, account.IsActive,
		account.UpdatedAt.Format(time.RFC3339Nano), account.ID,
	)
	if err != nil {
		return fmt.Errorf("actualizando cuenta %s: %w", account.ID, err)
	}
	return checkRowsAffected(res)
}

// Deactivate hace soft delete (RF-04): el registro y su historial de saldos
// asociados en account_balances se conservan intactos.
func (r *AccountRepository) Deactivate(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE accounts SET is_active = 0, updated_at = ? WHERE id = ?
	`, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("desactivando cuenta %s: %w", id, err)
	}
	return checkRowsAffected(res)
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+accountColumns+` FROM accounts WHERE id = ?`, id)
	acc, err := scanAccount(row)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, repository.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("buscando cuenta %s: %w", id, err)
	}
	return acc, nil
}

func (r *AccountRepository) List(ctx context.Context, filter repository.AccountFilter) ([]domain.Account, error) {
	query := "SELECT " + accountColumns + " FROM accounts WHERE 1 = 1"
	var args []interface{}

	if filter.ActiveOnly != nil && *filter.ActiveOnly {
		query += " AND is_active = 1"
	}
	if filter.Type != nil {
		query += " AND type = ?"
		args = append(args, string(*filter.Type))
	}
	if filter.Group != nil {
		query += " AND provider_group = ?"
		args = append(args, *filter.Group)
	}
	query += " ORDER BY name ASC"

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("listando cuentas: %w", err)
	}
	defer rows.Close()

	accounts := []domain.Account{}
	for rows.Next() {
		acc, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("leyendo cuenta: %w", err)
		}
		accounts = append(accounts, *acc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

// rowScanner abstrae *sql.Row y *sql.Rows, que comparten el método Scan
// pero no una interfaz común en la stdlib.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanAccount(s rowScanner) (*domain.Account, error) {
	var (
		id, name, accType, currency string
		providerGroup               sql.NullString
		isInvested, isActive        bool
		createdAt, updatedAt        string
	)
	if err := s.Scan(&id, &name, &accType, &currency, &providerGroup, &isInvested, &isActive, &createdAt, &updatedAt); err != nil {
		return nil, err
	}

	acc := &domain.Account{
		ID:         id,
		Name:       name,
		Type:       domain.AccountType(accType),
		Currency:   domain.Currency(currency),
		IsInvested: isInvested,
		IsActive:   isActive,
	}
	if providerGroup.Valid {
		v := providerGroup.String
		acc.ProviderGroup = &v
	}
	if t, err := time.Parse(time.RFC3339Nano, createdAt); err == nil {
		acc.CreatedAt = t
	}
	if t, err := time.Parse(time.RFC3339Nano, updatedAt); err == nil {
		acc.UpdatedAt = t
	}
	return acc, nil
}

func checkRowsAffected(res sql.Result) error {
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return repository.ErrNotFound
	}
	return nil
}

// isUniqueConstraintErr detecta violaciones de UNIQUE de forma independiente
// del driver (texto del error). Queda disponible para los próximos
// repositorios (snapshots, income_statements), que sí ejercitan RF-08 y
// RF-19 contra restricciones únicas.
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}
