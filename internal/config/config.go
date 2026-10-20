// Package config centralizes loading configuration from environment
// variables (and an optional .env file via godotenv), as defined in the
// Technical Plan section 2: no hardcoded configuration in the code.
package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config groups together all of the application's configuration.
type Config struct {
	// HTTPPort is the port the HTTP server listens on.
	HTTPPort string

	// DBDriver selects the database engine: "sqlite3" for now;
	// "postgres" / "mysql" are planned for when the migration happens (see
	// Technical Plan sections 1 and 10).
	DBDriver string
	// DBDSN is the data source name / connection string for the chosen
	// engine. For sqlite it's the path to the file, e.g. "./finance.db".
	DBDSN string

	// MigrationsPath points to the directory with the .up.sql/.down.sql files.
	MigrationsPath string

	// APIKey, if set, enables simple header-based authentication (RNF-01)
	// when the backend is exposed outside localhost. Empty = no
	// authentication (local use).
	APIKey string

	// DolarAPIBaseURL and related fields control which dollar exchange rate
	// is used by default, without touching code (Technical Plan section 5).
	DolarAPIBaseURL string
	DolarTipo       string // e.g. "oficial" (official rate)
	DolarCampo      string // e.g. "venta" (sell rate)

	CoinGeckoBaseURL string

	// ExternalAPITimeout limits how long to wait for external exchange rate
	// APIs before falling back to manual entry (Technical Plan section 5).
	ExternalAPITimeout time.Duration

	// GoogleServiceAccountFile is the path to the credentials JSON for
	// cmd/import (RF-09). Never committed to the repo.
	GoogleServiceAccountFile string
}

// Load reads a .env file if it exists (it's not an error if it's missing)
// and then builds the configuration from environment variables, applying
// sensible defaults for local development.
func Load() (*Config, error) {
	_ = godotenv.Load() // .env is optional; if it doesn't exist, proceed with the environment as-is

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
