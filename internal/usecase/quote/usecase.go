package quote

import (
	"context"

	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/quotes"
)

// UseCase orchestrates querying the external exchange rate APIs (RF-10)
// without blocking the user if one of them fails (Technical Plan section
// 5): each rate is resolved independently and errors travel alongside the
// response instead of aborting the whole request.
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
