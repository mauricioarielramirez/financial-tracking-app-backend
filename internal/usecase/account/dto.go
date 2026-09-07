// Package account contiene el usecase de cuentas/activos (RF-01 a RF-04):
// el orquestador entre el handler HTTP y el repository. Todo lo que entra o
// sale de este paquete lo hace en forma de DTO (Input/Output definidos acá
// mismo, a la misma altura que el usecase); domain.Account nunca cruza su
// frontera hacia afuera.
package account

import "time"

// CreateAccountInput es el contrato de entrada de POST /accounts.
// Lleva tags json porque ES el contrato de la API: el handler decodifica
// el body directamente sobre esta struct.
type CreateAccountInput struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Currency      string  `json:"currency"`
	ProviderGroup *string `json:"provider_group,omitempty"`
	IsInvested    bool    `json:"is_invested"`
}

// UpdateAccountInput es el contrato de entrada de PUT /accounts/{id}.
type UpdateAccountInput struct {
	Name          string  `json:"name"`
	Type          string  `json:"type"`
	Currency      string  `json:"currency"`
	ProviderGroup *string `json:"provider_group,omitempty"`
	IsInvested    bool    `json:"is_invested"`
}

// ListFilter es el contrato de entrada de GET /accounts (RF-25). Se resuelve
// acá (no en el handler ni en el repository) para no filtrar detalles del
// query string hacia el repository ni detalles de repository.AccountFilter
// hacia el handler.
type ListFilter struct {
	ActiveOnly *bool
	Type       *string
	Group      *string
}

// AccountOutput es el contrato de salida para cualquier operación que
// devuelva una cuenta (Create, Update, Get, y cada elemento de List).
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
