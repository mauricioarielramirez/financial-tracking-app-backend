// Package repository defines the persistence interfaces consumed by the
// service layer. No business logic or SQL embedded in handlers: all data
// access goes through these interfaces, which allows migrating from
// SQLite to Postgres/MySQL without touching the rest of the system (see
// Technical Plan section 1).
//
// Concrete implementations live in per-engine subpackages, e.g.
// internal/repository/sqlite.
//
// Mocks for these interfaces (used by internal/usecase/* tests) are
// generated once here and shared by every usecase that needs them.
// Requires mockgen installed (see README): go install
// go.uber.org/mock/mockgen@v0.6.0 && go generate ./...
package repository

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

import (
	"context"
	"errors"
	"time"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
)

// ErrNotFound is returned when an entity looked up by ID doesn't exist.
// Concrete implementations must map their specific error (e.g.
// sql.ErrNoRows) to this sentinel so the service layer doesn't depend on
// the database engine.
var ErrNotFound = errors.New("resource not found")

// ErrConflict is returned on unique constraint violations (e.g. two
// snapshots for the same month without an override, RF-08).
var ErrConflict = errors.New("conflict: resource already exists")

// AccountFilter groups the filters supported by GET /accounts (RF-25).
type AccountFilter struct {
	ActiveOnly *bool
	Type       *domain.AccountType
	Group      *string
}

// AccountRepository persists and queries accounts/assets (RF-01 to RF-04).
type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	Update(ctx context.Context, account *domain.Account) error
	// Deactivate does a soft delete: it doesn't erase the record, to
	// preserve the already-loaded balance history (RF-04).
	Deactivate(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.Account, error)
	List(ctx context.Context, filter AccountFilter) ([]domain.Account, error)
	// ExistsByName checks whether an account with that name already
	// exists, to prevent duplicates on create/edit. excludeID lets the
	// account being edited be excluded from the check (nil on create).
	ExistsByName(ctx context.Context, name string, excludeID *string) (bool, error)
}

// SnapshotFilter groups the GET /snapshots filters (RF-25).
type SnapshotFilter struct {
	From *time.Time
	To   *time.Time
}

// SnapshotRepository persists and queries monthly snapshots, their
// per-account balances, and the associated exchange rates (RF-05 to RF-09,
// RF-13 to RF-16).
type SnapshotRepository interface {
	// Create persists the snapshot along with its AccountBalance and
	// ExchangeRate records in a single transaction.
	Create(ctx context.Context, snapshot *domain.Snapshot, balances []domain.AccountBalance, rates []domain.ExchangeRate) error
	Update(ctx context.Context, snapshot *domain.Snapshot, balances []domain.AccountBalance, rates []domain.ExchangeRate) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.Snapshot, error)
	// ExistsForMonth checks whether a snapshot already exists for the given
	// date's month, to be able to warn about/block duplicates (RF-08).
	ExistsForMonth(ctx context.Context, date time.Time) (bool, error)
	List(ctx context.Context, filter SnapshotFilter) ([]domain.Snapshot, error)
	// GetBalances returns a snapshot's per-account breakdown (RF-21).
	GetBalances(ctx context.Context, snapshotID string) ([]domain.AccountBalance, error)
	// GetRates returns the exchange rates used in a snapshot (RF-12).
	GetRates(ctx context.Context, snapshotID string) ([]domain.ExchangeRate, error)
	// BalanceHistory returns an account's balance time series between two
	// dates (RF-23).
	BalanceHistory(ctx context.Context, accountID string, from, to time.Time) ([]domain.AccountBalance, error)
}

// IncomeStatementRepository persists and queries the monthly
// income/expense/savings record (RF-17 to RF-20).
type IncomeStatementRepository interface {
	Create(ctx context.Context, stmt *domain.IncomeStatement) error
	Update(ctx context.Context, stmt *domain.IncomeStatement) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.IncomeStatement, error)
	// GetByMonth looks up a specific month's record, used to detect
	// duplicates and to recalculate the cumulative total (RF-19).
	GetByMonth(ctx context.Context, periodMonth time.Time) (*domain.IncomeStatement, error)
	// ListOrderedByMonth returns every record in chronological order,
	// needed to recalculate the cumulative total in cascade.
	ListOrderedByMonth(ctx context.Context) ([]domain.IncomeStatement, error)
}
