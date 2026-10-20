// Package quote contains the suggested exchange rates usecase (RF-10,
// RF-11). It has no mapper: unlike account, there's no domain entity
// involved here (external exchange rates already arrive as
// decimal.Decimal from internal/quotes), so the Output is built directly
// in the usecase.
package quote

import "github.com/shopspring/decimal"

// SuggestedOutput is the response of GET /quotes/suggested: the client
// confirms or edits these values before creating the snapshot (RF-11).
type SuggestedOutput struct {
	USD          *decimal.Decimal           `json:"usd,omitempty"`
	USDErr       string                     `json:"usd_error,omitempty"`
	Crypto       map[string]decimal.Decimal `json:"crypto"`
	CryptoErrors map[string]string          `json:"crypto_errors,omitempty"`
}
