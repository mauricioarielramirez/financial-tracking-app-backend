package quote

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/quotes"
)

// UseCase orquesta la consulta a las APIs externas de cotización (RF-10)
// sin bloquear al usuario si alguna falla (Plan Técnico sección 5): cada
// cotización se resuelve de forma independiente y los errores viajan junto
// a la respuesta en vez de abortar todo el pedido.
type UseCase struct {
	provider quotes.QuoteProvider
}

func New(provider quotes.QuoteProvider) *UseCase {
	return &UseCase{provider: provider}
}

func (uc *UseCase) GetSuggested(ctx context.Context) SuggestedOutput {
	out := SuggestedOutput{
		Crypto:       map[string]decimal.Decimal{},
		CryptoErrors: map[string]string{},
	}

	usd, err := uc.provider.GetUSDRate(ctx)
	if err != nil {
		out.USDErr = err.Error()
	} else {
		out.USD = &usd
	}

	for _, symbol := range quotes.SupportedCryptos {
		rate, err := uc.provider.GetCryptoRate(ctx, symbol)
		if err != nil {
			out.CryptoErrors[symbol] = err.Error()
			continue
		}
		out.Crypto[symbol] = rate
	}

	return out
}
