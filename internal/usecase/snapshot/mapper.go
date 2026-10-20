package snapshot

import (
	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
)

//go:generate mockgen -source=mapper.go -destination=mapper_mock_test.go -package=snapshot

// Mapper translates domain.Snapshot/AccountBalance/ExchangeRate ⇄ this
// package's DTOs. It's an interface (not loose functions) so UseCase can
// be tested with a fake/mocked mapper, same as in internal/usecase/account.
type Mapper interface {
	// BalancesByAccount builds the accountID -> original balance map that
	// domain.BuildSnapshotDetail consumes, from what the client loaded
	// (RF-05).
	BalancesByAccount(in []BalanceInput) map[string]decimal.Decimal

	// ToDomainRates builds the snapshot's exchange rates (RF-06, RF-11).
	// It doesn't set ID/SnapshotID/FetchedAt: the usecase fills those in
	// before persisting.
	ToDomainRates(in []RateInput) []domain.ExchangeRate

	ToOutput(detail domain.SnapshotDetail) SnapshotOutput
	ToSummary(snap domain.Snapshot, consolidatedARS decimal.Decimal) SnapshotSummary
}

// mapper is Mapper's only real implementation. It has no state or
// dependencies, so a zero value (mapper{}) is enough.
type mapper struct{}

// NewMapper builds the real snapshot mapper.
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
