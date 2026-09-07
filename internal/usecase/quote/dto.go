// Package quote contiene el usecase de cotizaciones sugeridas (RF-10,
// RF-11). No tiene mapper: a diferencia de account, acá no hay una entidad
// de domain involucrada (las cotizaciones externas ya llegan como
// decimal.Decimal desde internal/quotes), así que el Output se arma
// directo en el usecase.
package quote

import "github.com/shopspring/decimal"

// SuggestedOutput es la respuesta de GET /quotes/suggested: el cliente
// confirma o edita estos valores antes de crear el snapshot (RF-11).
type SuggestedOutput struct {
	USD          *decimal.Decimal           `json:"usd,omitempty"`
	USDErr       string                     `json:"usd_error,omitempty"`
	Crypto       map[string]decimal.Decimal `json:"crypto"`
	CryptoErrors map[string]string          `json:"crypto_errors,omitempty"`
}
