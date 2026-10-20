package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// IncomeStatement is the monthly record of income, expenses, and savings,
// with a historical cumulative total (RF-17 to RF-20).
type IncomeStatement struct {
	ID                   string
	SnapshotID           *string
	PeriodMonth          time.Time // day 1 of the month
	IncomeTotalARS       decimal.Decimal
	ExpenseTotalARS      decimal.Decimal
	SavingsARS           decimal.Decimal
	SavingsPercentage    decimal.Decimal
	CumulativeSavingsARS decimal.Decimal
	IncomeTotalUSD       *decimal.Decimal
}

// Calculate derives savings, savings_percentage, and income_total_usd from
// the entered amounts and, if provided, the associated snapshot's USD
// exchange rate (RF-18, RF-20). It doesn't touch CumulativeSavingsARS: that
// requires knowing the previous periods and is resolved in the service
// layer (RF-19).
func (s *IncomeStatement) Calculate(usdRate *decimal.Decimal) {
	s.SavingsARS = s.IncomeTotalARS.Sub(s.ExpenseTotalARS)
	if s.IncomeTotalARS.IsPositive() {
		s.SavingsPercentage = s.SavingsARS.Div(s.IncomeTotalARS).Mul(decimal.NewFromInt(100))
	} else {
		s.SavingsPercentage = decimal.Zero
	}
	if usdRate != nil && usdRate.IsPositive() {
		usd := s.IncomeTotalARS.Div(*usdRate)
		s.IncomeTotalUSD = &usd
	}
}
