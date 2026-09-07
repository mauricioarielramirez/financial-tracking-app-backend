// Package domain contiene las entidades y reglas de negocio puras del sistema.
// No depende de HTTP, SQL concreto, JSON ni de ningún framework: son structs
// simples más las invariantes/validaciones propias del dominio (RF-01 a
// RF-04). Ni tags `json` ni `db`: la traducción a esos formatos es
// responsabilidad de quien los consume (mapper en internal/usecase/*, o el
// propio repository en internal/repository/sqlite), nunca del dominio.
package domain

import (
	"errors"
	"time"
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
	ErrAccountNameRequired     = errors.New("el nombre de la cuenta es obligatorio")
	ErrAccountTypeInvalid      = errors.New("tipo de cuenta inválido")
	ErrAccountCurrencyRequired = errors.New("la moneda de la cuenta es obligatoria")
)

// Validate aplica las invariantes mínimas de una cuenta (RF-02).
// Se llama tanto en alta como en edición, antes de persistir.
func (a *Account) Validate() error {
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
