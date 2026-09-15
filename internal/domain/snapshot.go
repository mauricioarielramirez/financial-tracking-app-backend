package domain

import (
	"errors"
	"fmt"
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

var (
	// ErrSnapshotMissingBalance se devuelve cuando falta el saldo de alguna
	// cuenta activa al armar un snapshot (RF-05: "saldo de cada cuenta
	// activa a esa fecha").
	ErrSnapshotMissingBalance = errors.New("falta el saldo de una o más cuentas activas")
	// ErrSnapshotMissingRate se devuelve cuando una cuenta está en una
	// moneda distinta de ARS y no hay cotización provista para convertirla
	// (RF-06, RF-13).
	ErrSnapshotMissingRate = errors.New("falta la cotización de una o más monedas usadas por las cuentas activas")
)

// BuildSnapshotDetail calcula, a partir de las cuentas activas, sus saldos
// originales y las cotizaciones del snapshot, el desglose por cuenta
// convertido a ARS, la posición consolidada y la clasificación
// invertido/no invertido (RF-13 a RF-16). Es una función pura de dominio:
// no persiste nada, sólo arma los valores que luego el usecase de snapshot
// pasa al repository.
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
			return SnapshotDetail{}, fmt.Errorf("%w: cuenta %s", ErrSnapshotMissingBalance, acc.Name)
		}

		balanceARS := original
		if acc.Currency != CurrencyARS {
			rate, ok := rateByCurrency[acc.Currency]
			if !ok {
				return SnapshotDetail{}, fmt.Errorf("%w: moneda %s (cuenta %s)", ErrSnapshotMissingRate, acc.Currency, acc.Name)
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

	// RF-15: el porcentaje de cada cuenta se calcula en una segunda pasada,
	// una vez conocido el total consolidado.
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

// SummarizeStoredBalances reconstruye los totales de un snapshot ya
// persistido (posición consolidada en ARS/USD e invertido/no invertido, RF-14
// a RF-16) a partir de los AccountBalance y ExchangeRate ya guardados. A
// diferencia de BuildSnapshotDetail, no recalcula BalanceARS ni
// PercentageOfTotal: los toma tal cual quedaron guardados, para no violar la
// trazabilidad histórica (RNF-02). investedByAccount clasifica cada cuenta
// por su estado actual de "invertido" (puede diferir del que tenía al
// momento del snapshot si la cuenta fue editada después).
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
