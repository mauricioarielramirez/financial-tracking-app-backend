// Package repository define las interfaces de persistencia que consume la
// capa de servicio. Ninguna lógica de negocio ni SQL embebido en handlers:
// todo acceso a datos pasa por estas interfaces, lo que permite migrar de
// SQLite a Postgres/MySQL sin tocar el resto del sistema (ver Plan Técnico
// sección 1).
//
// Las implementaciones concretas viven en subpaquetes por motor, p. ej.
// internal/repository/sqlite.
//
// Los mocks de estas interfaces (usados por los tests de internal/usecase/*)
// se generan una sola vez acá y se comparten entre todos los usecases que
// los necesiten. Requiere mockgen instalado (ver README): go install
// go.uber.org/mock/mockgen@v0.6.0 && go generate ./...
package repository

//go:generate mockgen -source=repository.go -destination=mocks/repository_mock.go -package=mocks

import (
	"context"
	"errors"
	"time"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
)

// ErrNotFound se devuelve cuando una entidad buscada por ID no existe.
// Las implementaciones concretas deben mapear su error específico
// (p. ej. sql.ErrNoRows) a este sentinel para que la capa de servicio no
// dependa del motor de base de datos.
var ErrNotFound = errors.New("recurso no encontrado")

// ErrConflict se devuelve ante violaciones de restricciones únicas
// (p. ej. dos snapshots para el mismo mes sin override, RF-08).
var ErrConflict = errors.New("conflicto: el recurso ya existe")

// AccountFilter agrupa los filtros soportados por GET /accounts (RF-25).
type AccountFilter struct {
	ActiveOnly *bool
	Type       *domain.AccountType
	Group      *string
}

// AccountRepository persiste y consulta cuentas/activos (RF-01 a RF-04).
type AccountRepository interface {
	Create(ctx context.Context, account *domain.Account) error
	Update(ctx context.Context, account *domain.Account) error
	// Deactivate hace soft delete: no borra el registro para preservar el
	// historial de saldos ya cargados (RF-04).
	Deactivate(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.Account, error)
	List(ctx context.Context, filter AccountFilter) ([]domain.Account, error)
}

// SnapshotFilter agrupa los filtros de GET /snapshots (RF-25).
type SnapshotFilter struct {
	From *time.Time
	To   *time.Time
}

// SnapshotRepository persiste y consulta snapshots mensuales, sus saldos por
// cuenta y las cotizaciones asociadas (RF-05 a RF-09, RF-13 a RF-16).
type SnapshotRepository interface {
	// Create persiste el snapshot junto con sus AccountBalance y
	// ExchangeRate en una única transacción.
	Create(ctx context.Context, snapshot *domain.Snapshot, balances []domain.AccountBalance, rates []domain.ExchangeRate) error
	Update(ctx context.Context, snapshot *domain.Snapshot, balances []domain.AccountBalance, rates []domain.ExchangeRate) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.Snapshot, error)
	// ExistsForMonth verifica si ya existe un snapshot para el mes de la
	// fecha dada, para poder advertir/bloquear duplicados (RF-08).
	ExistsForMonth(ctx context.Context, date time.Time) (bool, error)
	List(ctx context.Context, filter SnapshotFilter) ([]domain.Snapshot, error)
	// GetBalances devuelve el desglose por cuenta de un snapshot (RF-21).
	GetBalances(ctx context.Context, snapshotID string) ([]domain.AccountBalance, error)
	// GetRates devuelve las cotizaciones usadas en un snapshot (RF-12).
	GetRates(ctx context.Context, snapshotID string) ([]domain.ExchangeRate, error)
	// BalanceHistory devuelve la serie temporal de saldos de una cuenta
	// entre dos fechas (RF-23).
	BalanceHistory(ctx context.Context, accountID string, from, to time.Time) ([]domain.AccountBalance, error)
}

// IncomeStatementRepository persiste y consulta el registro mensual de
// ingresos/gastos/ahorro (RF-17 a RF-20).
type IncomeStatementRepository interface {
	Create(ctx context.Context, stmt *domain.IncomeStatement) error
	Update(ctx context.Context, stmt *domain.IncomeStatement) error
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*domain.IncomeStatement, error)
	// GetByMonth busca el registro de un mes puntual, usado para detectar
	// duplicados y para recalcular el acumulado (RF-19).
	GetByMonth(ctx context.Context, periodMonth time.Time) (*domain.IncomeStatement, error)
	// ListOrderedByMonth devuelve todos los registros ordenados
	// cronológicamente, necesario para recalcular el acumulado en cascada.
	ListOrderedByMonth(ctx context.Context) ([]domain.IncomeStatement, error)
}
