package quotes

import (
	"context"
	"strings"

	"github.com/shopspring/decimal"
)

// CompositeProvider combina un proveedor de USD y uno de cripto detrás de
// una única implementación de QuoteProvider, que es lo que consume el
// resto del sistema (Plan Técnico sección 5).
type CompositeProvider struct {
	usdProvider    QuoteProvider
	cryptoProvider QuoteProvider
}

func NewCompositeProvider(usdProvider, cryptoProvider QuoteProvider) *CompositeProvider {
	return &CompositeProvider{usdProvider: usdProvider, cryptoProvider: cryptoProvider}
}

var _ QuoteProvider = (*CompositeProvider)(nil)

func (c *CompositeProvider) GetUSDRate(ctx context.Context) (decimal.Decimal, error) {
	return c.usdProvider.GetUSDRate(ctx)
}

func (c *CompositeProvider) GetCryptoRate(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return c.cryptoProvider.GetCryptoRate(ctx, strings.ToUpper(symbol))
}
