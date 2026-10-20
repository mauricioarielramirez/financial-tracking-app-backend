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

// cryptoIDs maps our supported symbols to the id expected by the CoinGecko
// API (RF-10, Functional Requirements section 4).
var cryptoIDs = map[string]string{
	"BTC":  "bitcoin",
	"ETH":  "ethereum",
	"DAI":  "dai",
	"USDT": "tether",
}

// CoinGeckoClient queries CoinGecko's public API (free tier, no key) for
// crypto exchange rates in ARS.
type CoinGeckoClient struct {
	BaseURL string // e.g. "https://api.coingecko.com/api/v3"
	Client  *http.Client
}

func NewCoinGeckoClient(baseURL string, timeout time.Duration) *CoinGeckoClient {
	return &CoinGeckoClient{
		BaseURL: baseURL,
		Client:  &http.Client{Timeout: timeout},
	}
}

// GetUSDRate: CoinGeckoClient only resolves crypto; see CompositeProvider.
func (c *CoinGeckoClient) GetUSDRate(ctx context.Context) (decimal.Decimal, error) {
	return decimal.Zero, fmt.Errorf("CoinGeckoClient does not provide a USD/ARS exchange rate; use DolarAPIClient")
}

// GetCryptoRate implements quotes.QuoteProvider for crypto symbols.
func (c *CoinGeckoClient) GetCryptoRate(ctx context.Context, symbol string) (decimal.Decimal, error) {
	id, ok := cryptoIDs[strings.ToUpper(symbol)]
	if !ok {
		return decimal.Zero, fmt.Errorf("unsupported crypto symbol: %q", symbol)
	}

	endpoint := fmt.Sprintf("%s/simple/price?ids=%s&vs_currencies=ars",
		c.BaseURL, url.QueryEscape(id))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return decimal.Zero, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("querying CoinGecko: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("CoinGecko returned status %d", resp.StatusCode)
	}

	// Response shaped like: {"bitcoin": {"ars": 123456.0}}
	var body map[string]map[string]float64
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return decimal.Zero, fmt.Errorf("decoding CoinGecko response: %w", err)
	}

	coin, ok := body[id]
	if !ok {
		return decimal.Zero, fmt.Errorf("CoinGecko returned no data for %q", id)
	}
	rate, ok := coin["ars"]
	if !ok {
		return decimal.Zero, fmt.Errorf("CoinGecko returned no ARS rate for %q", id)
	}

	return decimal.NewFromFloat(rate), nil
}
