package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// DolarAPIClient queries https://dolarapi.com for the dollar exchange rate
// (RF-06, RF-10). The dollar type (oficial/blue/...) and the field
// (compra/venta, i.e. buy/sell) are configurable (Technical Plan section 5)
// so they can be changed without touching code.
type DolarAPIClient struct {
	BaseURL string // e.g. "https://dolarapi.com/v1"
	Tipo    string // e.g. "oficial" (official rate)
	Campo   string // "compra" (buy) | "venta" (sell)
	Client  *http.Client
}

// NewDolarAPIClient builds the client with a timeout matching the
// configured one (Technical Plan section 5: don't block the user if the
// external API is slow).
func NewDolarAPIClient(baseURL, tipo, campo string, timeout time.Duration) *DolarAPIClient {
	return &DolarAPIClient{
		BaseURL: baseURL,
		Tipo:    tipo,
		Campo:   campo,
		Client:  &http.Client{Timeout: timeout},
	}
}

type dolarAPIResponse struct {
	Moneda             string  `json:"moneda"`
	Casa               string  `json:"casa"`
	Compra             float64 `json:"compra"`
	Venta              float64 `json:"venta"`
	FechaActualizacion string  `json:"fechaActualizacion"`
}

// GetUSDRate implements quotes.QuoteProvider.
func (c *DolarAPIClient) GetUSDRate(ctx context.Context) (decimal.Decimal, error) {
	url := fmt.Sprintf("%s/dolares/%s", c.BaseURL, c.Tipo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("querying DolarAPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("DolarAPI returned status %d", resp.StatusCode)
	}

	var body dolarAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return decimal.Zero, fmt.Errorf("decoding DolarAPI response: %w", err)
	}

	var value float64
	switch c.Campo {
	case "compra":
		value = body.Compra
	default:
		value = body.Venta
	}

	return decimal.NewFromFloat(value), nil
}

// GetCryptoRate: DolarAPIClient only resolves USD; the rest is delegated to
// CoinGeckoClient. It returns an explicit error if called anyway, so it's
// clear at runtime that the correct provider needs to be used (see
// CompositeProvider below).
func (c *DolarAPIClient) GetCryptoRate(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return decimal.Zero, fmt.Errorf("DolarAPIClient does not provide crypto rates (symbol %q); use CoinGeckoClient", symbol)
}
