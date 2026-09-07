package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// Snapshot es el relevamiento mensual de saldos de todas las cuentas activas,
// junto con las cotizaciones vigentes usadas en ese momento (RF-05 a RF-09).
type Snapshot struct {
	ID           string
	SnapshotDate time.Time
	Notes        *string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// ExchangeRate es la cotización de una moneda/cripto respecto al ARS,
// asociada e inmutable a un snapshot puntual (RF-10 a RF-12).
type ExchangeRate struct {
	ID           string
	SnapshotID   string
	CurrencyCode Currency
	RateToARS    decimal.Decimal
	Source       string
	FetchedAt    time.Time
}

// Fuentes posibles para una cotización (RF-06, RF-11).
const (
	QuoteSourceDolarAPI    = "api:dolarapi"
	QuoteSourceCoinGecko   = "api:coingecko"
	QuoteSourceManual      = "manual"
	QuoteSourceImportSheet = "import:google_sheets"
)

// AccountBalance es el saldo de una cuenta puntual dentro de un snapshot,
// junto con los valores calculados (RF-05, RF-13, RF-15).
type AccountBalance struct {
	ID                string
	SnapshotID        string
	AccountID         string
	BalanceOriginal   decimal.Decimal
	BalanceARS        decimal.Decimal
	PercentageOfTotal decimal.Decimal
}

// SnapshotDetail agrega un snapshot con su desglose completo por cuenta y las
// cotizaciones usadas — es la forma en la que se arma GET /snapshots/{id}
// (RF-21). No se persiste como tal: la arma el usecase a partir de otras
// entidades de dominio (nunca es en sí mismo un DTO, sigue siendo dominio;
// el mapper de internal/usecase/snapshot es quien la traduce a su Output).
type SnapshotDetail struct {
	Snapshot            Snapshot
	Balances            []AccountBalance
	Rates               []ExchangeRate
	ConsolidatedARS     decimal.Decimal
	ConsolidatedUSD     *decimal.Decimal
	InvestedTotalARS    decimal.Decimal
	NotInvestedTotalARS decimal.Decimal
}
