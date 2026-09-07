package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// cryptoIDs mapea nuestros símbolos soportados al id que espera la API de
// CoinGecko (RF-10, Requerimientos Funcionales sección 4).
var cryptoIDs = map[string]string{
	"BTC":  "bitcoin",
	"ETH":  "ethereum",
	"DAI":  "dai",
	"USDT": "tether",
}

// CoinGeckoClient consulta la API pública (nivel gratuito, sin key) de
// CoinGecko para cotizaciones cripto en ARS.
type CoinGeckoClient struct {
	BaseURL string // p. ej. "https://api.coingecko.com/api/v3"
	Client  *http.Client
}

func NewCoinGeckoClient(baseURL string, timeout time.Duration) *CoinGeckoClient {
	return &CoinGeckoClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: timeout},
	}
}

// GetUSDRate: CoinGeckoClient sólo resuelve cripto; ver CompositeProvider.
func (c *CoinGeckoClient) GetUSDRate(ctx context.Context) (decimal.Decimal, error) {
	return decimal.Zero, fmt.Errorf("CoinGeckoClient no provee cotización de USD/ARS; usar DolarAPIClient")
}

// GetCryptoRate implementa quotes.QuoteProvider para símbolos cripto.
func (c *CoinGeckoClient) GetCryptoRate(ctx context.Context, symbol string) (decimal.Decimal, error) {
	id, ok := cryptoIDs[strings.ToUpper(symbol)]
	if !ok {
		return decimal.Zero, fmt.Errorf("símbolo cripto no soportado: %q", symbol)
	}

	endpoint := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=ars",
		c.BaseURL, url.QueryEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return decimal.Zero, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("consultando CoinGecko: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("CoinGecko devolvió status %d", resp.StatusCode)
	}

	// Respuesta con forma: {"bitcoin": {"ars": 123456.0}}
	var body map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return decimal.Zero, fmt.Errorf("decodificando respuesta de CoinGecko: %w", err)
	}

	coin, ok := body[id]
	if !ok {
		return decimal.Zero, fmt.Errorf("CoinGecko no devolvió datos para %q", id)
	}
	rate, ok := coin["ars"]
	if !ok {
		return decimal.Zero, fmt.Errorf("CoinGecko no devolvió cotización en ARS para %q", id)
	}

	return decimal.NewFromFloat(rate), nil
}
