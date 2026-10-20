// Package domain contiene las entidades y reglas de negocio puras del sistema.
// No depende de HTTP, SQL concreto, JSON ni de ningún framework: son structs
// simples más las invariantes/validaciones propias del dominio (RF-01 a
// RF-04). Ni tags `json` ni `db`: la traducción a esos formatos es
// responsabilidad de quien los consume (mapper en internal/usecase/*, o el
// propio repository en internal/repository/sqlite), nunca del dominio.
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

// AccountType enumera los tipos de cuenta/activo soportados (RF-02).
type AccountType string

const (
	AccountTypeBank           AccountType = "bank"
	AccountTypeWallet         AccountType = "wallet"
	AccountTypeFixedTerm      AccountType = "fixed_term"
	AccountTypeInvestmentFund AccountType = "investment_fund"
	AccountTypeCash           AccountType = "cash"
	AccountTypeCrypto         AccountType = "crypto"
	AccountTypeOther          AccountType = "other"
)

// ValidAccountTypes lista los tipos aceptados, usada para validar altas/ediciones.
var ValidAccountTypes = map[AccountType]bool{
	AccountTypeBank:           true,
	AccountTypeWallet:         true,
	AccountTypeFixedTerm:      true,
	AccountTypeInvestmentFund: true,
	AccountTypeCash:           true,
	AccountTypeCrypto:         true,
	AccountTypeOther:          true,
}

// Currency identifica la moneda de origen de una cuenta (RF-02).
type Currency string

const (
	CurrencyARS  Currency = "ARS"
	CurrencyUSD  Currency = "USD"
	CurrencyBTC  Currency = "BTC"
	CurrencyETH  Currency = "ETH"
	CurrencyDAI  Currency = "DAI"
	CurrencyUSDT Currency = "USDT"
)

// Account representa una cuenta/activo financiero definido dinámicamente
// por el usuario (RF-01 a RF-04).
type Account struct {
	ID            string
	Name          string
	Type          AccountType
	Currency      Currency
	ProviderGroup *string
	IsInvested    bool
	IsActive      bool
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

var (
	ErrAccountNameRequired     = errors.New("account name is required")
	ErrAccountTypeInvalid      = errors.New("invalid account type")
	ErrAccountCurrencyRequired = errors.New("account currency is required")
)

// normalizeAccountName estandariza el nombre a Title Case (cada palabra con
// su primera letra en mayúscula), p. ej. "cuenta de mercadopago ars" ->
// "Cuenta De Mercadopago Ars". Las palabras que ya llegan en mayúsculas
// (acrónimos o códigos de moneda escritos a mano, p. ej. "ARS", "USD") se
// dejan intactas en lugar de forzarlas a minúscula.
func normalizeAccountName(name string) string {
	words := strings.Fields(name)
	for i, w := range words {
		if isAllUpper(w) {
			continue
		}
		runes := []rune(strings.ToLower(w))
		runes[0] = unicode.ToUpper(runes[0])
		words[i] = string(runes)
	}
	return strings.Join(words, " ")
}

// isAllUpper es true si w tiene al menos una letra y ninguna está en
// minúscula (así se detectan acrónimos/códigos de moneda escritos a mano).
func isAllUpper(w string) bool {
	hasLetter := false
	for _, r := range w {
		if unicode.IsLower(r) {
			return false
		}
		if unicode.IsLetter(r) {
			hasLetter = true
		}
	}
	return hasLetter
}

// Validate normaliza el nombre a Title Case y aplica las invariantes
// mínimas de una cuenta (RF-02). Se llama tanto en alta como en edición,
// antes de persistir.
func (a *Account) Validate() error {
	a.Name = normalizeAccountName(a.Name)

	if a.Name == "" {
		return ErrAccountNameRequired
	}
	if !ValidAccountTypes[a.Type] {
		return ErrAccountTypeInvalid
	}
	if a.Currency == "" {
		return ErrAccountCurrencyRequired
	}
	return nil
}
