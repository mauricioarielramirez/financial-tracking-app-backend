package sqlite_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository/sqlite"
)

// newTestSnapshotRepo abre una base SQLite temporal (un archivo por test),
// le aplica las migraciones reales del proyecto y crea una cuenta de
// prueba: los saldos requieren una cuenta existente por la FK de
// account_balances.
func newTestSnapshotRepo(t *testing.T) (*sqlite.SnapshotRepository, *domain.Account) {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	db, err := sqlite.Open(dbPath)
	if err != nil {
		t.Fatalf("abriendo sqlite de prueba: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	migrationsPath, err := filepath.Abs(filepath.Join("..", "..", "..", "migrations"))
	if err != nil {
		t.Fatalf("resolviendo ruta de migraciones: %v", err)
	}
	if err := sqlite.Migrate(context.Background(), db, migrationsPath); err != nil {
		t.Fatalf("aplicando migraciones: %v", err)
	}

	accountRepo := sqlite.NewAccountRepository(db)
	acc := &domain.Account{Name: "Cuenta ARS", Type: domain.AccountTypeBank, Currency: domain.CurrencyARS}
	if err := accountRepo.Create(context.Background(), acc); err != nil {
		t.Fatalf("creando cuenta de prueba: %v", err)
	}

	return sqlite.NewSnapshotRepository(db), acc
}

func TestSnapshotRepository_CreateGetByIDRoundTrip(t *testing.T) {
	repo, acc := newTestSnapshotRepo(t)
	ctx := context.Background()

	snap := &domain.Snapshot{SnapshotDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)}
	balances := []domain.AccountBalance{
		{AccountID: acc.ID, BalanceOriginal: decimal.NewFromInt(1000), BalanceARS: decimal.NewFromInt(1000), PercentageOfTotal: decimal.NewFromInt(100)},
	}
	rates := []domain.ExchangeRate{
		{CurrencyCode: domain.CurrencyUSD, RateToARS: decimal.NewFromInt(1000), Source: domain.QuoteSourceManual},
	}

	if err := repo.Create(ctx, snap, balances, rates); err != nil {
		t.Fatalf("creando snapshot: %v", err)
	}
	if snap.ID == "" {
		t.Fatal("se esperaba que Create asignara un ID")
	}

	got, err := repo.GetByID(ctx, snap.ID)
	if err != nil {
		t.Fatalf("buscando snapshot: %v", err)
	}
	if !got.SnapshotDate.Equal(snap.SnapshotDate) {
		t.Fatalf("fecha inesperada: %v", got.SnapshotDate)
	}

	gotBalances, err := repo.GetBalances(ctx, snap.ID)
	if err != nil {
		t.Fatalf("buscando saldos: %v", err)
	}
	if len(gotBalances) != 1 || !gotBalances[0].BalanceARS.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("saldos inesperados: %+v", gotBalances)
	}

	gotRates, err := repo.GetRates(ctx, snap.ID)
	if err != nil {
		t.Fatalf("buscando cotizaciones: %v", err)
	}
	if len(gotRates) != 1 || !gotRates[0].RateToARS.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("cotizaciones inesperadas: %+v", gotRates)
	}
}

// TestSnapshotRepository_ExistsForMonth cubre RF-08.
func TestSnapshotRepository_ExistsForMonth(t *testing.T) {
	repo, acc := newTestSnapshotRepo(t)
	ctx := context.Background()

	snap := &domain.Snapshot{SnapshotDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)}
	balances := []domain.AccountBalance{
		{AccountID: acc.ID, BalanceOriginal: decimal.NewFromInt(100), BalanceARS: decimal.NewFromInt(100), PercentageOfTotal: decimal.NewFromInt(100)},
	}
	if err := repo.Create(ctx, snap, balances, nil); err != nil {
		t.Fatalf("creando snapshot: %v", err)
	}

	exists, err := repo.ExistsForMonth(ctx, time.Date(2024, 3, 15, 12, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("verificando mes: %v", err)
	}
	if !exists {
		t.Fatal("se esperaba que ExistsForMonth devolviera true para marzo 2024")
	}

	exists, err = repo.ExistsForMonth(ctx, time.Date(2024, 4, 1, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("verificando mes: %v", err)
	}
	if exists {
		t.Fatal("no se esperaba un snapshot en abril 2024")
	}
}

// TestSnapshotRepository_UpdateReemplazaSaldosYCotizaciones cubre RF-07.
func TestSnapshotRepository_UpdateReemplazaSaldosYCotizaciones(t *testing.T) {
	repo, acc := newTestSnapshotRepo(t)
	ctx := context.Background()

	snap := &domain.Snapshot{SnapshotDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)}
	balances := []domain.AccountBalance{
		{AccountID: acc.ID, BalanceOriginal: decimal.NewFromInt(1000), BalanceARS: decimal.NewFromInt(1000), PercentageOfTotal: decimal.NewFromInt(100)},
	}
	if err := repo.Create(ctx, snap, balances, nil); err != nil {
		t.Fatalf("creando snapshot: %v", err)
	}

	notes := "actualizado"
	snap.Notes = &notes
	updatedBalances := []domain.AccountBalance{
		{AccountID: acc.ID, BalanceOriginal: decimal.NewFromInt(2000), BalanceARS: decimal.NewFromInt(2000), PercentageOfTotal: decimal.NewFromInt(100)},
	}
	if err := repo.Update(ctx, snap, updatedBalances, nil); err != nil {
		t.Fatalf("actualizando snapshot: %v", err)
	}

	gotBalances, err := repo.GetBalances(ctx, snap.ID)
	if err != nil {
		t.Fatalf("buscando saldos tras update: %v", err)
	}
	if len(gotBalances) != 1 || !gotBalances[0].BalanceARS.Equal(decimal.NewFromInt(2000)) {
		t.Fatalf("saldos tras update inesperados: %+v", gotBalances)
	}
}

// TestSnapshotRepository_DeleteBorraEnCascada cubre el soporte de RF-07/09
// para corregir cargas: eliminar un snapshot no debe dejar huérfanos en
// account_balances/exchange_rates.
func TestSnapshotRepository_DeleteBorraEnCascada(t *testing.T) {
	repo, acc := newTestSnapshotRepo(t)
	ctx := context.Background()

	snap := &domain.Snapshot{SnapshotDate: time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC)}
	balances := []domain.AccountBalance{
		{AccountID: acc.ID, BalanceOriginal: decimal.NewFromInt(100), BalanceARS: decimal.NewFromInt(100), PercentageOfTotal: decimal.NewFromInt(100)},
	}
	if err := repo.Create(ctx, snap, balances, nil); err != nil {
		t.Fatalf("creando snapshot: %v", err)
	}

	if err := repo.Delete(ctx, snap.ID); err != nil {
		t.Fatalf("eliminando snapshot: %v", err)
	}
	if _, err := repo.GetByID(ctx, snap.ID); err != repository.ErrNotFound {
		t.Fatalf("se esperaba repository.ErrNotFound tras eliminar, se obtuvo: %v", err)
	}
	gotBalances, err := repo.GetBalances(ctx, snap.ID)
	if err != nil {
		t.Fatalf("buscando saldos tras eliminar: %v", err)
	}
	if len(gotBalances) != 0 {
		t.Fatalf("se esperaban 0 saldos tras eliminar en cascada, se obtuvieron %d", len(gotBalances))
	}
}
