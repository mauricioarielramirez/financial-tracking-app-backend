// Package config centraliza la carga de configuración desde variables de
// entorno (y un archivo .env opcional vía godotenv), tal como define el Plan
// Técnico sección 2: nada de configuración hardcodeada en el código.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config agrupa toda la configuración de la aplicación.
type Config struct {
	// HTTPPort es el puerto en el que escucha el servidor HTTP.
	HTTPPort string

	// DBDriver selecciona el motor de base de datos: "sqlite3" por ahora;
	// "postgres" / "mysql" quedan previstos para cuando se migre (ver Plan
	// Técnico sección 1 y 10).
	DBDriver string
	// DBDSN es el data source name / connection string del motor elegido.
	// Para sqlite es la ruta al archivo, p. ej. "./finance.db".
	DBDSN string

	// MigrationsPath apunta al directorio con los archivos .up.sql/.down.sql.
	MigrationsPath string

	// APIKey, si se define, habilita autenticación simple por header
	// (RNF-01) cuando el backend se expone fuera de localhost. Vacío = sin
	// autenticación (uso local).
	APIKey string

	// DolarAPIBaseURL y campos relacionados controlan qué cotización de
	// dólar se usa por defecto, sin tocar código (Plan Técnico sección 5).
	DolarAPIBaseURL string
	DolarTipo       string // p. ej. "oficial"
	DolarCampo      string // p. ej. "venta"

	CoinGeckoBaseURL string

	// ExternalAPITimeout limita cuánto se espera a las APIs externas de
	// cotización antes de degradar a carga manual (Plan Técnico sección 5).
	ExternalAPITimeout time.Duration

	// GoogleServiceAccountFile es la ruta al JSON de credenciales para
	// cmd/import (RF-09). Nunca se versiona en el repo.
	GoogleServiceAccountFile string
}

// Load lee un archivo .env si existe (no es un error que falte) y luego
// arma la configuración a partir de variables de entorno, aplicando
// defaults razonables para desarrollo local.
func Load() (*Config, error) {
	_ = godotenv.Load() // .env es opcional; si no existe, se sigue con el entorno tal cual

	cfg := &Config{
		HTTPPort:                 getEnv("HTTP_PORT", "8080"),
		DBDriver:                 getEnv("DB_DRIVER", "sqlite3"),
		DBDSN:                    getEnv("DB_DSN", "./finance.db"),
		MigrationsPath:           getEnv("MIGRATIONS_PATH", "./migrations"),
		APIKey:                   getEnv("API_KEY", ""),
		DolarAPIBaseURL:          getEnv("DOLARAPI_BASE_URL", "https://dolarapi.com/v1"),
		DolarTipo:                getEnv("DOLAR_TIPO", "oficial"),
		DolarCampo:               getEnv("DOLAR_CAMPO", "venta"),
		CoinGeckoBaseURL:         getEnv("COINGECKO_BASE_URL", "https://api.coingecko.com/api/v3"),
		GoogleServiceAccountFile: getEnv("GOOGLE_SERVICE_ACCOUNT_FILE", ""),
	}

	timeoutSeconds, err := strconv.Atoi(getEnv("EXTERNAL_API_TIMEOUT_SECONDS", "5"))
	if err != nil {
		timeoutSeconds = 5
	}
	cfg.ExternalAPITimeout = time.Duration(timeoutSeconds) * time.Second

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}
