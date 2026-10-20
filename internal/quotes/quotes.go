// Package quotes encapsulates access to the external exchange rate APIs
// (RF-06, RF-10, RF-11). The QuoteProvider interface is the only thing the
// rest of the system knows about: it allows swapping providers (or mocking
// in tests) without touching services or handlers (Technical Plan section 5).
package quotes

import (
	"context"

	"github.com/shopspring/decimal"
)

// SupportedCryptos are the crypto symbols covered in this first version
// (see Functional Requirements, section 4).
var SupportedCryptos = []string{"BTC", "ETH", "DAI", "USDT"}

// QuoteProvider groups fetching suggested exchange rates. An implementation
// can delegate to one or more external APIs (DolarAPI for USD, CoinGecko
// for crypto) or, in tests, return fixed values.
type QuoteProvider interface {
	// GetUSDRate returns the suggested ARS-per-1-USD rate (RF-06).
	GetUSDRate(ctx context.Context) (decimal.Decimal, error)
	// GetCryptoRate returns the ARS rate for 1 unit of the given symbol
	// (e.g. "BTC"). symbol must be one of SupportedCryptos.
	GetCryptoRate(ctx context.Context, symbol string) (decimal.Decimal, error)
}
