package snapshot

import (
	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
)

//go:generate mockgen -source=mapper.go -destination=mapper_mock_test.go -package=snapshot

// Mapper traduce domain.Snapshot/AccountBalance/ExchangeRate ⇄ los DTO de
// este paquete. Es una interfaz (y no funciones sueltas) para que UseCase
// pueda testearse con un mapper falso/mockeado, igual que en
// internal/usecase/account.
type Mapper interface {
	// BalancesByAccount arma el mapa accountID -> saldo original que
	// consume domain.BuildSnapshotDetail, a partir de lo cargado por el
	// cliente (RF-05).
	BalancesByAccount(in []BalanceInput) map[string]decimal.Decimal

	// ToDomainRates arma las cotizaciones del snapshot (RF-06, RF-11). No
	// setea ID/SnapshotID/FetchedAt: esos los completa el usecase antes de
	// persistir.
	ToDomainRates(in []RateInput) []domain.ExchangeRate

	ToOutput(detail domain.SnapshotDetail) SnapshotOutput
	ToSummary(snap domain.Snapshot, consolidatedARS decimal.Decimal) SnapshotSummary
}

// mapper es la única implementación real de Mapper. No tiene estado ni
// dependencias, así que un valor cero (mapper{}) alcanza.
type mapper struct{}

// NewMapper construye el mapper real de snapshots.
func NewMapper() Mapper {
	return mapper{}
}

func (mapper) BalancesByAccount(in []BalanceInput) map[string]decimal.Decimal {
	out := make(map[string]decimal.Decimal, len(in))
	for _, b := range in {
		out[b.AccountID] = b.Balance
	}
	return out
}

func (mapper) ToDomainRates(in []RateInput) []domain.ExchangeRate {
	out := make([]domain.ExchangeRate, len(in))
	for i, r := range in {
		source := r.Source
		if source == "" {
			source = domain.QuoteSourceManual
		}
		out[i] = domain.ExchangeRate{
			CurrencyCode: domain.Currency(r.CurrencyCode),
			RateToARS:    r.RateToARS,
			Source:       source,
		}
	}
	return out
}

func (mapper) ToOutput(detail domain.SnapshotDetail) SnapshotOutput {
	balances := make([]AccountBalanceOutput, len(detail.Balances))
	for i, b := range detail.Balances {
		balances[i] = AccountBalanceOutput{
			AccountID:         b.AccountID,
			BalanceOriginal:   b.BalanceOriginal,
			BalanceARS:        b.BalanceARS,
			PercentageOfTotal: b.PercentageOfTotal,
		}
	}

	rates := make([]ExchangeRateOutput, len(detail.Rates))
	for i, r := range detail.Rates {
		rates[i] = ExchangeRateOutput{
			CurrencyCode: string(r.CurrencyCode),
			RateToARS:    r.RateToARS,
			Source:       r.Source,
		}
	}

	return SnapshotOutput{
		ID:                  detail.Snapshot.ID,
		SnapshotDate:        detail.Snapshot.SnapshotDate,
		Notes:               detail.Snapshot.Notes,
		Balances:            balances,
		Rates:               rates,
		ConsolidatedARS:     detail.ConsolidatedARS,
		ConsolidatedUSD:     detail.ConsolidatedUSD,
		InvestedTotalARS:    detail.InvestedTotalARS,
		NotInvestedTotalARS: detail.NotInvestedTotalARS,
		CreatedAt:           detail.Snapshot.CreatedAt,
		UpdatedAt:           detail.Snapshot.UpdatedAt,
	}
}

func (mapper) ToSummary(snap domain.Snapshot, consolidatedARS decimal.Decimal) SnapshotSummary {
	return SnapshotSummary{
		ID:              snap.ID,
		SnapshotDate:    snap.SnapshotDate,
		Notes:           snap.Notes,
		ConsolidatedARS: consolidatedARS,
	}
}
