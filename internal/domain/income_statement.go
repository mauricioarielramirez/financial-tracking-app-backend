package domain

import (
	"time"

	"github.com/shopspring/decimal"
)

// IncomeStatement es el registro mensual de ingreso, gasto y ahorro,
// con acumulado histórico (RF-17 a RF-20).
type IncomeStatement struct {
	ID                   string
	SnapshotID           *string
	PeriodMonth          time.Time // día 1 del mes
	IncomeTotalARS       decimal.Decimal
	ExpenseTotalARS      decimal.Decimal
	SavingsARS           decimal.Decimal
	SavingsPercentage    decimal.Decimal
	CumulativeSavingsARS decimal.Decimal
	IncomeTotalUSD       *decimal.Decimal
}

// Calculate deriva savings, savings_percentage e income_total_usd a partir de
// los montos cargados y, si se provee, la cotización USD del snapshot
// asociado (RF-18, RF-20). No toca CumulativeSavingsARS: eso requiere
// conocer los períodos anteriores y se resuelve en la capa de servicio
// (RF-19).
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
