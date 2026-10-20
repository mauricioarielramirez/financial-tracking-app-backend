// Package snapshot contains the monthly snapshots usecase (RF-05 to
// RF-09, RF-11 to RF-16): the orchestrator between the HTTP handler and
// the snapshot and account repositories. Everything that enters or leaves
// this package does so as a DTO; domain.Snapshot/AccountBalance/
// ExchangeRate never cross its boundary.
package snapshot

import (
	"time"

	"github.com/shopspring/decimal"
)

// BalanceInput is an account's original (unconverted) balance within a
// snapshot (RF-05).
type BalanceInput struct {
	AccountID string          `json:"account_id"`
	Balance   decimal.Decimal `json:"balance"`
}

// RateInput is an exchange rate provided by the client when creating or
// editing a snapshot (RF-06, RF-11): it can come from GET
// /quotes/suggested's suggestion or be entered by hand. Source is left
// empty when the client doesn't specify it; the usecase fills in
// domain.QuoteSourceManual by default.
type RateInput struct {
	CurrencyCode string          `json:"currency_code"`
	RateToARS    decimal.Decimal `json:"rate_to_ars"`
	Source       string          `json:"source,omitempty"`
}

// CreateSnapshotInput is the input contract for POST /snapshots.
type CreateSnapshotInput struct {
	SnapshotDate time.Time      `json:"snapshot_date"`
	Notes        *string        `json:"notes,omitempty"`
	Balances     []BalanceInput `json:"balances"`
	Rates        []RateInput    `json:"rates,omitempty"`
	// Force allows confirming the entry even if a snapshot already exists
	// for the same month (RF-08).
	Force bool `json:"force,omitempty"`
}

// UpdateSnapshotInput is the input contract for PUT /snapshots/{id}
// (RF-07): replaces the snapshot's date, notes, balances, and exchange
// rates.
type UpdateSnapshotInput struct {
	SnapshotDate time.Time      `json:"snapshot_date"`
	Notes        *string        `json:"notes,omitempty"`
	Balances     []BalanceInput `json:"balances"`
	Rates        []RateInput    `json:"rates,omitempty"`
	// Force allows confirming the date change even if the new month
	// already has another snapshot loaded (RF-08).
	Force bool `json:"force,omitempty"`
}

// ImportItem is a historical snapshot within a bulk import (RF-09). It
// doesn't carry Force: an import can always coexist with several
// snapshots per month, since it reconstructs a full historical series.
type ImportItem struct {
	SnapshotDate time.Time      `json:"snapshot_date"`
	Notes        *string        `json:"notes,omitempty"`
	Balances     []BalanceInput `json:"balances"`
	Rates        []RateInput    `json:"rates,omitempty"`
}

// ImportInput is the input contract for POST /snapshots/import (RF-09).
type ImportInput struct {
	Items []ImportItem `json:"items"`
}

// ImportItemResult reports, for each item in the import, whether it was
// created successfully or the reason it failed; a failed item doesn't
// abort the rest.
type ImportItemResult struct {
	Index        int     `json:"index"`
	SnapshotID   *string `json:"snapshot_id,omitempty"`
	SnapshotDate string  `json:"snapshot_date"`
	Error        string  `json:"error,omitempty"`
}

// ImportOutput is the response of POST /snapshots/import.
type ImportOutput struct {
	Results []ImportItemResult `json:"results"`
}

// ListFilter is the input contract for GET /snapshots (RF-25).
type ListFilter struct {
	From *time.Time
	To   *time.Time
}

// AccountBalanceOutput is the per-account breakdown within a snapshot
// (RF-13, RF-15).
type AccountBalanceOutput struct {
	AccountID         string          `json:"account_id"`
	BalanceOriginal   decimal.Decimal `json:"balance_original"`
	BalanceARS        decimal.Decimal `json:"balance_ars"`
	PercentageOfTotal decimal.Decimal `json:"percentage_of_total"`
}

// ExchangeRateOutput is an exchange rate used in the snapshot (RF-12).
type ExchangeRateOutput struct {
	CurrencyCode string          `json:"currency_code"`
	RateToARS    decimal.Decimal `json:"rate_to_ars"`
	Source       string          `json:"source"`
}

// SnapshotOutput is the output contract for Create, Update, and GetByID:
// the snapshot with its full breakdown (RF-13 to RF-16, RF-21, RF-22).
type SnapshotOutput struct {
	ID                  string                 `json:"id"`
	SnapshotDate        time.Time              `json:"snapshot_date"`
	Notes               *string                `json:"notes,omitempty"`
	Balances            []AccountBalanceOutput `json:"balances"`
	Rates               []ExchangeRateOutput   `json:"rates"`
	ConsolidatedARS     decimal.Decimal        `json:"consolidated_ars"`
	ConsolidatedUSD     *decimal.Decimal       `json:"consolidated_usd,omitempty"`
	InvestedTotalARS    decimal.Decimal        `json:"invested_total_ars"`
	NotInvestedTotalARS decimal.Decimal        `json:"not_invested_total_ars"`
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// SnapshotSummary is the summarized shape used by GET /snapshots (without
// the full per-account breakdown, so as not to overload the listing).
type SnapshotSummary struct {
	ID              string          `json:"id"`
	SnapshotDate    time.Time       `json:"snapshot_date"`
	Notes           *string         `json:"notes,omitempty"`
	ConsolidatedARS decimal.Decimal `json:"consolidated_ars"`
}
