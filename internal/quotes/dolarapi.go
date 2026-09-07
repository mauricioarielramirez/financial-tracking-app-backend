package quotes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/shopspring/decimal"
)

// DolarAPIClient consulta https://dolarapi.com para la cotización del dólar
// (RF-06, RF-10). El tipo de dólar (oficial/blue/...) y el campo
// (compra/venta) son configurables (Plan Técnico sección 5) para poder
// cambiarlos sin tocar código.
type DolarAPIClient struct {
	BaseURL string // p. ej. "https://dolarapi.com/v1"
	Tipo    string // p. ej. "oficial"
	Campo   string // "compra" | "venta"
	Client  *http.Client
}

// NewDolarAPIClient arma el cliente con un timeout acorde al configurado
// (Plan Técnico sección 5: no bloquear al usuario si la API externa tarda).
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

// GetUSDRate implementa quotes.QuoteProvider.
func (c *DolarAPIClient) GetUSDRate(ctx context.Context) (decimal.Decimal, error) {
	url := fmt.Sprintf("%s/dolares/%s", c.BaseURL, c.Tipo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return decimal.Zero, err
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return decimal.Zero, fmt.Errorf("consultando DolarAPI: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return decimal.Zero, fmt.Errorf("DolarAPI devolvió status %d", resp.StatusCode)
	}

	var body dolarAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return decimal.Zero, fmt.Errorf("decodificando respuesta de DolarAPI: %w", err)
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

// GetCryptoRate: DolarAPIClient sólo resuelve USD; delega el resto a
// CoinGeckoClient. Devuelve un error explícito si se lo llama igual, para
// que quede claro en tiempo de ejecución que hay que usar el proveedor
// correcto (ver CompositeProvider más abajo).
func (c *DolarAPIClient) GetCryptoRate(ctx context.Context, symbol string) (decimal.Decimal, error) {
	return decimal.Zero, fmt.Errorf("DolarAPIClient no provee cotizaciones cripto (símbolo %q); usar CoinGeckoClient", symbol)
}
