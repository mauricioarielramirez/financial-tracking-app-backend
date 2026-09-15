package domain_test

import (
	"errors"
	"testing"

	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
)

func TestBuildSnapshotDetail_ConvierteAARSYCalculaPorcentajes(t *testing.T) {
	accounts := []domain.Account{
		{ID: "a1", Name: "Banco ARS", Currency: domain.CurrencyARS, IsInvested: false},
		{ID: "a2", Name: "Cuenta USD", Currency: domain.CurrencyUSD, IsInvested: true},
	}
	balances := map[string]decimal.Decimal{
		"a1": decimal.NewFromInt(1000),
		"a2": decimal.NewFromInt(10),
	}
	rates := []domain.ExchangeRate{
		{CurrencyCode: domain.CurrencyUSD, RateToARS: decimal.NewFromInt(1000), Source: domain.QuoteSourceManual},
	}

	detail, err := domain.BuildSnapshotDetail(domain.Snapshot{ID: "snap-1"}, accounts, balances, rates)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}

	// Consolidado: 1000 ARS + (10 USD * 1000) = 11000 ARS.
	if !detail.ConsolidatedARS.Equal(decimal.NewFromInt(11000)) {
		t.Fatalf("consolidado ARS inesperado: %s", detail.ConsolidatedARS)
	}
	if !detail.InvestedTotalARS.Equal(decimal.NewFromInt(10000)) {
		t.Fatalf("total invertido inesperado: %s", detail.InvestedTotalARS)
	}
	if !detail.NotInvestedTotalARS.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("total no invertido inesperado: %s", detail.NotInvestedTotalARS)
	}
	if detail.ConsolidatedUSD == nil || !detail.ConsolidatedUSD.Equal(decimal.NewFromInt(11)) {
		t.Fatalf("consolidado USD inesperado: %v", detail.ConsolidatedUSD)
	}

	var pctByAccount = map[string]decimal.Decimal{}
	for _, b := range detail.Balances {
		pctByAccount[b.AccountID] = b.PercentageOfTotal
		if b.SnapshotID != "snap-1" {
			t.Fatalf("SnapshotID no propagado en el balance de %s", b.AccountID)
		}
	}
	// 1000 / 11000 * 100 ≈ 9.0909...
	if !pctByAccount["a1"].Round(2).Equal(decimal.RequireFromString("9.09")) {
		t.Fatalf("porcentaje de a1 inesperado: %s", pctByAccount["a1"])
	}
	// 10000 / 11000 * 100 ≈ 90.9090...
	if !pctByAccount["a2"].Round(2).Equal(decimal.RequireFromString("90.91")) {
		t.Fatalf("porcentaje de a2 inesperado: %s", pctByAccount["a2"])
	}
}

func TestBuildSnapshotDetail_FaltaSaldoDeCuentaActiva(t *testing.T) {
	accounts := []domain.Account{{ID: "a1", Name: "Banco ARS", Currency: domain.CurrencyARS}}

	_, err := domain.BuildSnapshotDetail(domain.Snapshot{}, accounts, map[string]decimal.Decimal{}, nil)
	if !errors.Is(err, domain.ErrSnapshotMissingBalance) {
		t.Fatalf("se esperaba ErrSnapshotMissingBalance, se obtuvo: %v", err)
	}
}

func TestBuildSnapshotDetail_FaltaCotizacionParaMonedaNoARS(t *testing.T) {
	accounts := []domain.Account{{ID: "a1", Name: "Cuenta USD", Currency: domain.CurrencyUSD}}
	balances := map[string]decimal.Decimal{"a1": decimal.NewFromInt(10)}

	_, err := domain.BuildSnapshotDetail(domain.Snapshot{}, accounts, balances, nil)
	if !errors.Is(err, domain.ErrSnapshotMissingRate) {
		t.Fatalf("se esperaba ErrSnapshotMissingRate, se obtuvo: %v", err)
	}
}

func TestSummarizeStoredBalances_NoRecalculaSoloSuma(t *testing.T) {
	balances := []domain.AccountBalance{
		{AccountID: "a1", BalanceARS: decimal.NewFromInt(1000)},
		{AccountID: "a2", BalanceARS: decimal.NewFromInt(2000)},
	}
	rates := []domain.ExchangeRate{
		{CurrencyCode: domain.CurrencyUSD, RateToARS: decimal.NewFromInt(1000)},
	}
	investedByAccount := map[string]bool{"a1": false, "a2": true}

	consolidatedARS, consolidatedUSD, investedARS, notInvestedARS := domain.SummarizeStoredBalances(balances, rates, investedByAccount)

	if !consolidatedARS.Equal(decimal.NewFromInt(3000)) {
		t.Fatalf("consolidado ARS inesperado: %s", consolidatedARS)
	}
	if consolidatedUSD == nil || !consolidatedUSD.Equal(decimal.NewFromInt(3)) {
		t.Fatalf("consolidado USD inesperado: %v", consolidatedUSD)
	}
	if !investedARS.Equal(decimal.NewFromInt(2000)) {
		t.Fatalf("invertido inesperado: %s", investedARS)
	}
	if !notInvestedARS.Equal(decimal.NewFromInt(1000)) {
		t.Fatalf("no invertido inesperado: %s", notInvestedARS)
	}
}
