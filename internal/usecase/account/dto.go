// Package account contains the accounts/assets usecase (RF-01 to RF-04):
// the orchestrator between the HTTP handler and the repository. Everything
// that enters or leaves this package does so as a DTO (Input/Output
// defined right here, alongside the usecase); domain.Account never
// crosses its boundary.
package account

import "time"

// CreateAccountInput is the input contract for POST /accounts.
// It carries json tags because it IS the API contract: the handler decodes
// the body directly onto this struct.
type CreateAccountInput struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Currency      string  `json:"currency"`
	ProviderGroup *string `json:"provider_group,omitempty"`
	IsInvested    bool    `json:"is_invested"`
}

// UpdateAccountInput is the input contract for PUT /accounts/{id}.
type UpdateAccountInput struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Currency      string  `json:"currency"`
	ProviderGroup *string `json:"provider_group,omitempty"`
	IsInvested    bool    `json:"is_invested"`
}

// PatchAccountInput is the input contract for PATCH /accounts/{id}: unlike
// UpdateAccountInput (a full PUT replace), every field is a pointer to
// distinguish "wasn't in the JSON" (nil, the current value is kept) from
// "came with a value" (that specific field is overwritten).
type PatchAccountInput struct {
	Name          *string `json:"name,omitempty"`
	Type          *string `json:"type,omitempty"`
	Currency      *string `json:"currency,omitempty"`
	ProviderGroup *string `json:"provider_group,omitempty"`
	IsInvested    *bool   `json:"is_invested,omitempty"`
}

// ListFilter is the input contract for GET /accounts (RF-25). It's
// resolved here (not in the handler or the repository) so query string
// details don't leak into the repository, nor repository.AccountFilter
// details into the handler.
type ListFilter struct {
	ActiveOnly *bool
	Type       *string
	Group      *string
}

// AccountOutput is the output contract for any operation that returns an
// account (Create, Update, Get, and each element of List).
type AccountOutput struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Type          string    `json:"type"`
	Currency      string    `json:"currency"`
	ProviderGroup *string   `json:"provider_group,omitempty"`
	IsInvested    bool      `json:"is_invested"`
	IsActive      bool      `json:"is_active"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}
