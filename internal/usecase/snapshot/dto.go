// Package snapshot contiene el usecase de snapshots mensuales (RF-05 a
// RF-09, RF-11 a RF-16): el orquestador entre el handler HTTP y los
// repositories de snapshot y de cuentas. Todo lo que entra o sale de este
// paquete lo hace en forma de DTO; domain.Snapshot/AccountBalance/
// ExchangeRate nunca cruzan su frontera hacia afuera.
package snapshot

import (
	"time"

	"github.com/shopspring/decimal"
)

// BalanceInput es el saldo original (sin convertir) de una cuenta dentro de
// un snapshot (RF-05).
type BalanceInput struct {
	AccountID string          `json:"account_id"`
	Balance   decimal.Decimal `json:"balance"`
}

// RateInput es una cotización provista por el cliente al crear o editar un
// snapshot (RF-06, RF-11): puede venir de la sugerencia de
// GET /quotes/suggested o cargada a mano. Source queda vacío cuando el
// cliente no lo indica; el usecase completa domain.QuoteSourceManual por
// defecto.
type RateInput struct {
	CurrencyCode string          `json:"currency_code"`
	RateToARS    decimal.Decimal `json:"rate_to_ars"`
	Source       string          `json:"source,omitempty"`
}

// CreateSnapshotInput es el contrato de entrada de POST /snapshots.
type CreateSnapshotInput struct {
	SnapshotDate time.Time      `json:"snapshot_date"`
	Notes        *string        `json:"notes,omitempty"`
	Balances     []BalanceInput `json:"balances"`
	Rates        []RateInput    `json:"rates,omitempty"`
	// Force permite confirmar la carga aunque ya exista un snapshot en el
	// mismo mes (RF-08).
	Force bool `json:"force,omitempty"`
}

// UpdateSnapshotInput es el contrato de entrada de PUT /snapshots/{id}
// (RF-07): reemplaza fecha, notas, saldos y cotizaciones del snapshot.
type UpdateSnapshotInput struct {
	SnapshotDate time.Time      `json:"snapshot_date"`
	Notes        *string        `json:"notes,omitempty"`
	Balances     []BalanceInput `json:"balances"`
	Rates        []RateInput    `json:"rates,omitempty"`
	// Force permite confirmar el cambio de fecha aunque el nuevo mes ya
	// tenga otro snapshot cargado (RF-08).
	Force bool `json:"force,omitempty"`
}

// ImportItem es un snapshot histórico dentro de una carga masiva (RF-09).
// No lleva Force: la importación siempre puede convivir con varios
// snapshots por mes, ya que reconstruye una serie histórica completa.
type ImportItem struct {
	SnapshotDate time.Time      `json:"snapshot_date"`
	Notes        *string        `json:"notes,omitempty"`
	Balances     []BalanceInput `json:"balances"`
	Rates        []RateInput    `json:"rates,omitempty"`
}

// ImportInput es el contrato de entrada de POST /snapshots/import (RF-09).
type ImportInput struct {
	Items []ImportItem `json:"items"`
}

// ImportItemResult informa, por cada elemento de la importación, si se creó
// correctamente o el motivo del error; un item fallido no aborta el resto.
type ImportItemResult struct {
	Index        int     `json:"index"`
	SnapshotID   *string `json:"snapshot_id,omitempty"`
	SnapshotDate string  `json:"snapshot_date"`
	Error        string  `json:"error,omitempty"`
}

// ImportOutput es la respuesta de POST /snapshots/import.
type ImportOutput struct {
	Results []ImportItemResult `json:"results"`
}

// ListFilter es el contrato de entrada de GET /snapshots (RF-25).
type ListFilter struct {
	From *time.Time
	To   *time.Time
}

// AccountBalanceOutput es el desglose por cuenta dentro de un snapshot
// (RF-13, RF-15).
type AccountBalanceOutput struct {
	AccountID         string          `json:"account_id"`
	BalanceOriginal   decimal.Decimal `json:"balance_original"`
	BalanceARS        decimal.Decimal `json:"balance_ars"`
	PercentageOfTotal decimal.Decimal `json:"percentage_of_total"`
}

// ExchangeRateOutput es una cotización usada en el snapshot (RF-12).
type ExchangeRateOutput struct {
	CurrencyCode string          `json:"currency_code"`
	RateToARS    decimal.Decimal `json:"rate_to_ars"`
	Source       string          `json:"source"`
}

// SnapshotOutput es el contrato de salida de Create, Update y GetByID: el
// snapshot con su desglose completo (RF-13 a RF-16, RF-21, RF-22).
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

// SnapshotSummary es la forma resumida usada por GET /snapshots (sin el
// desglose completo por cuenta, para no sobrecargar el listado).
type SnapshotSummary struct {
	ID              string          `json:"id"`
	SnapshotDate    time.Time       `json:"snapshot_date"`
	Notes           *string         `json:"notes,omitempty"`
	ConsolidatedARS decimal.Decimal `json:"consolidated_ars"`
}
