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

// AccountRepository implements repository.AccountRepository over SQLite,
// using database/sql directly (no query builder) to avoid adding extra
// dependencies.
type AccountRepository struct {
	db *sql.DB
}

// NewAccountRepository builds the account repository.
func NewAccountRepository(db *sql.DB) *AccountRepository {
	return &AccountRepository{db: db}
}

// compile-time check: AccountRepository must satisfy the domain interface.
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
		account.IsActive = true // a new account is active by default (RF-01)
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
		return fmt.Errorf("inserting account: %w", err)
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
		return fmt.Errorf("updating account %s: %w", account.ID, err)
	}
	return checkRowsAffected(res)
}

// Deactivate does a soft delete (RF-04): the record and its associated
// balance history in account_balances are kept intact.
func (r *AccountRepository) Deactivate(ctx context.Context, id string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE accounts SET is_active = 0, updated_at = ? WHERE id = ?
	`, time.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return fmt.Errorf("deactivating account %s: %w", id, err)
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
		return nil, fmt.Errorf("fetching account %s: %w", id, err)
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
		return nil, fmt.Errorf("listing accounts: %w", err)
	}
	defer rows.Close()

	accounts := []domain.Account{}
	for rows.Next() {
		acc, err := scanAccount(rows)
		if err != nil {
			return nil, fmt.Errorf("reading account: %w", err)
		}
		accounts = append(accounts, *acc)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return accounts, nil
}

// ExistsByName checks whether an account with that name already exists
// (RF-01/RF-02). excludeID is used on edit so the account isn't compared
// against itself.
func (r *AccountRepository) ExistsByName(ctx context.Context, name string, excludeID *string) (bool, error) {
	query := "SELECT 1 FROM accounts WHERE name = ?"
	args := []interface{}{name}
	if excludeID != nil {
		query += " AND id != ?"
		args = append(args, *excludeID)
	}
	query += " LIMIT 1"

	var exists int
	err := r.db.QueryRowContext(ctx, query, args...).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("checking account name %q: %w", name, err)
	}
	return true, nil
}

// rowScanner abstracts *sql.Row and *sql.Rows, which share the Scan method
// but no common interface in the stdlib.
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

// isUniqueConstraintErr detects UNIQUE violations in a driver-independent
// way (by matching the error text). It's left available for the upcoming
// repositories (snapshots, income_statements), which do exercise RF-08 and
// RF-19 against unique constraints.
func isUniqueConstraintErr(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique") || strings.Contains(msg, "constraint")
}
