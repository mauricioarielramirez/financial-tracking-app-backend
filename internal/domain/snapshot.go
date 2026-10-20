package domain

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// Snapshot is the monthly reading of all active accounts' balances, along
// with the exchange rates in effect at that moment (RF-05 to RF-09).
type Snapshot struct {
	ID           string
	SnapshotDate time.Time
	Notes        *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ExchangeRate is a currency/crypto's exchange rate against ARS, tied to and
// immutable for a specific snapshot (RF-10 to RF-12).
type ExchangeRate struct {
	ID           string
	SnapshotID   string
	CurrencyCode Currency
	RateToARS    decimal.Decimal
	Source       string
	FetchedAt    time.Time
}

// Possible sources for an exchange rate (RF-06, RF-11).
const (
	QuoteSourceDolarAPI    = "api:dolarapi"
	QuoteSourceCoinGecko   = "api:coingecko"
	QuoteSourceManual      = "manual"
	QuoteSourceImportSheet = "import:google_sheets"
)

// AccountBalance is a specific account's balance within a snapshot, along
// with the calculated values (RF-05, RF-13, RF-15).
type AccountBalance struct {
	ID                string
	SnapshotID        string
	AccountID         string
	BalanceOriginal   decimal.Decimal
	BalanceARS        decimal.Decimal
	PercentageOfTotal decimal.Decimal
}

// SnapshotDetail bundles a snapshot with its full per-account breakdown and
// the exchange rates used — this is how GET /snapshots/{id} is assembled
// (RF-21). It isn't persisted as such: the usecase builds it from other
// domain entities (it's never a DTO itself, it stays domain; the mapper in
// internal/usecase/snapshot is what translates it to its Output).
type SnapshotDetail struct {
	Snapshot            Snapshot
	Balances            []AccountBalance
	Rates               []ExchangeRate
	ConsolidatedARS     decimal.Decimal
	ConsolidatedUSD     *decimal.Decimal
	InvestedTotalARS    decimal.Decimal
	NotInvestedTotalARS decimal.Decimal
}

var (
	// ErrSnapshotMissingBalance is returned when some active account's
	// balance is missing while building a snapshot (RF-05: "each active
	// account's balance as of that date").
	ErrSnapshotMissingBalance = errors.New("missing balance for one or more active accounts")
	// ErrSnapshotMissingRate is returned when an account is in a currency
	// other than ARS and no exchange rate was provided to convert it
	// (RF-06, RF-13).
	ErrSnapshotMissingRate = errors.New("missing exchange rate for one or more currencies used by active accounts")
)

// BuildSnapshotDetail calculates, from the active accounts, their original
// balances, and the snapshot's exchange rates, the per-account breakdown
// converted to ARS, the consolidated position, and the invested/not-invested
// classification (RF-13 to RF-16). It's a pure domain function: it doesn't
// persist anything, it just builds the values that the snapshot usecase
// then passes to the repository.
func BuildSnapshotDetail(snap Snapshot, accounts []Account, balancesByAccount map[string]decimal.Decimal, rates []ExchangeRate) (SnapshotDetail, error) {
	rateByCurrency := make(map[Currency]decimal.Decimal, len(rates))
	for _, r := range rates {
		rateByCurrency[r.CurrencyCode] = r.RateToARS
	}

	balances := make([]AccountBalance, 0, len(accounts))
	consolidatedARS := decimal.Zero
	investedARS := decimal.Zero
	notInvestedARS := decimal.Zero

	for _, acc := range accounts {
		original, ok := balancesByAccount[acc.ID]
		if !ok {
			return SnapshotDetail{}, fmt.Errorf("%w: account %s", ErrSnapshotMissingBalance, acc.Name)
		}

		balanceARS := original
		if acc.Currency != CurrencyARS {
			rate, ok := rateByCurrency[acc.Currency]
			if !ok {
				return SnapshotDetail{}, fmt.Errorf("%w: currency %s (account %s)", ErrSnapshotMissingRate, acc.Currency, acc.Name)
			}
			balanceARS = original.Mul(rate)
		}

		balances = append(balances, AccountBalance{
			SnapshotID:      snap.ID,
			AccountID:       acc.ID,
			BalanceOriginal: original,
			BalanceARS:      balanceARS,
		})

		consolidatedARS = consolidatedARS.Add(balanceARS)
		if acc.IsInvested {
			investedARS = investedARS.Add(balanceARS)
		} else {
			notInvestedARS = notInvestedARS.Add(balanceARS)
		}
	}

	// RF-15: each account's percentage is calculated in a second pass, once
	// the consolidated total is known.
	for i := range balances {
		if consolidatedARS.IsPositive() {
			balances[i].PercentageOfTotal = balances[i].BalanceARS.Div(consolidatedARS).Mul(decimal.NewFromInt(100))
		} else {
			balances[i].PercentageOfTotal = decimal.Zero
		}
	}

	detail := SnapshotDetail{
		Snapshot:            snap,
		Balances:            balances,
		Rates:               rates,
		ConsolidatedARS:     consolidatedARS,
		InvestedTotalARS:    investedARS,
		NotInvestedTotalARS: notInvestedARS,
	}

	if usdRate, ok := rateByCurrency[CurrencyUSD]; ok && usdRate.IsPositive() {
		usd := consolidatedARS.Div(usdRate)
		detail.ConsolidatedUSD = &usd
	}

	return detail, nil
}

// SummarizeStoredBalances reconstructs the totals for an already-persisted
// snapshot (consolidated position in ARS/USD and invested/not-invested,
// RF-14 to RF-16) from the already-saved AccountBalance and ExchangeRate
// records. Unlike BuildSnapshotDetail, it doesn't recalculate BalanceARS or
// PercentageOfTotal: it takes them exactly as they were saved, so as not to
// violate historical traceability (RNF-02). investedByAccount classifies
// each account by its current "invested" status (which may differ from
// what it had at the time of the snapshot if the account was edited since).
func SummarizeStoredBalances(balances []AccountBalance, rates []ExchangeRate, investedByAccount map[string]bool) (consolidatedARS decimal.Decimal, consolidatedUSD *decimal.Decimal, investedARS, notInvestedARS decimal.Decimal) {
	consolidatedARS = decimal.Zero
	investedARS = decimal.Zero
	notInvestedARS = decimal.Zero

	for _, b := range balances {
		consolidatedARS = consolidatedARS.Add(b.BalanceARS)
		if investedByAccount[b.AccountID] {
			investedARS = investedARS.Add(b.BalanceARS)
		} else {
			notInvestedARS = notInvestedARS.Add(b.BalanceARS)
		}
	}

	for _, r := range rates {
		if r.CurrencyCode == CurrencyUSD && r.RateToARS.IsPositive() {
			usd := consolidatedARS.Div(r.RateToARS)
			consolidatedUSD = &usd
			break
		}
	}

	return consolidatedARS, consolidatedUSD, investedARS, notInvestedARS
}
