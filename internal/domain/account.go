// Package domain contains the system's pure entities and business rules.
// It doesn't depend on HTTP, any concrete SQL, JSON, or any framework: just
// plain structs plus the domain's own invariants/validations (RF-01 to
// RF-04). No `json` or `db` tags: translating to those formats is the
// responsibility of whoever consumes them (a mapper in internal/usecase/*,
// or the repository itself in internal/repository/sqlite), never the
// domain's.
package domain

import (
	"errors"
	"strings"
	"time"
	"unicode"
)

// AccountType enumerates the supported account/asset types (RF-02).
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

// ValidAccountTypes lists the accepted types, used to validate creates/edits.
var ValidAccountTypes = map[AccountType]bool{
	AccountTypeBank:           true,
	AccountTypeWallet:         true,
	AccountTypeFixedTerm:      true,
	AccountTypeInvestmentFund: true,
	AccountTypeCash:           true,
	AccountTypeCrypto:         true,
	AccountTypeOther:          true,
}

// Currency identifies an account's source currency (RF-02).
type Currency string

const (
	CurrencyARS  Currency = "ARS"
	CurrencyUSD  Currency = "USD"
	CurrencyBTC  Currency = "BTC"
	CurrencyETH  Currency = "ETH"
	CurrencyDAI  Currency = "DAI"
	CurrencyUSDT Currency = "USDT"
)

// Account represents a financial account/asset dynamically defined by the
// user (RF-01 to RF-04).
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

// normalizeAccountName standardizes the name to Title Case (each word with
// its first letter capitalized), e.g. "cuenta de mercadopago ars" ->
// "Cuenta De Mercadopago Ars". Words that already arrive fully uppercase
// (acronyms or currency codes typed by hand, e.g. "ARS", "USD") are left
// untouched instead of being forced to lowercase.
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

// isAllUpper is true if w has at least one letter and none of them are
// lowercase (this is how hand-typed acronyms/currency codes are detected).
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

// Validate normalizes the name to Title Case and applies an account's
// minimum invariants (RF-02). It's called on both create and edit, before
// persisting.
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
