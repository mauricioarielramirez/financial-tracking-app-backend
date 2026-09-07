// Package quotes encapsula el acceso a las APIs externas de cotización
// (RF-06, RF-10, RF-11). La interfaz QuoteProvider es lo único que conoce
// el resto del sistema: permite reemplazar de proveedor (o mockear en
// tests) sin tocar servicios ni handlers (Plan Técnico sección 5).
package quotes

import (
	"context"

	"github.com/shopspring/decimal"
)

// SupportedCryptos son los símbolos cripto contemplados en esta primera
// versión (ver Requerimientos Funcionales, sección 4).
var SupportedCryptos = []string{"BTC", "ETH", "DAI", "USDT"}

// QuoteProvider agrupa la obtención de cotizaciones sugeridas. Una
// implementación puede delegar en una o más APIs externas (DolarAPI para
// USD, CoinGecko para cripto) o, en tests, devolver valores fijos.
type QuoteProvider interface {
	// GetUSDRate devuelve la cotización ARS por 1 USD sugerida (RF-06).
	GetUSDRate(ctx context.Context) (decimal.Decimal, error)
	// GetCryptoRate devuelve la cotización ARS por 1 unidad del símbolo
	// dado (p. ej. "BTC"). symbol debe ser uno de SupportedCryptos.
	GetCryptoRate(ctx context.Context, symbol string) (decimal.Decimal, error)
}
