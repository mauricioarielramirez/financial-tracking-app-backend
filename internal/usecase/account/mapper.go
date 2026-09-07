package account

import "github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"

//go:generate mockgen -source=mapper.go -destination=mapper_mock_test.go -package=account

// Mapper traduce domain.Account ⇄ los DTO de este paquete. Es una interfaz
// (y no funciones sueltas) para que UseCase pueda testearse con un mapper
// falso/mockeado sin ejecutar la lógica real de mapeo, y para que el mapper
// real pueda testearse por su cuenta.
type Mapper interface {
	// ToDomain arma una cuenta nueva a partir del alta (RF-01). No setea
	// ID/IsActive/CreatedAt/UpdatedAt: esos los completa el repository al
	// persistir.
	ToDomain(in CreateAccountInput) domain.Account

	// ApplyUpdate aplica los campos editables de UpdateAccountInput sobre
	// una cuenta ya existente, sin tocar ID, IsActive ni los timestamps.
	ApplyUpdate(in UpdateAccountInput, acc *domain.Account)

	ToOutput(a domain.Account) AccountOutput
	ToOutputList(accounts []domain.Account) []AccountOutput
}

// mapper es la única implementación real de Mapper. No tiene estado ni
// dependencias, así que un valor cero (mapper{}) alcanza.
type mapper struct{}

// NewMapper construye el mapper real de cuentas.
func NewMapper() Mapper {
	return mapper{}
}

func (mapper) ToDomain(in CreateAccountInput) domain.Account {
	return domain.Account{
		Name:          in.Name,
		Type:          domain.AccountType(in.Type),
		Currency:      domain.Currency(in.Currency),
		ProviderGroup: in.ProviderGroup,
		IsInvested:    in.IsInvested,
	}
}

func (mapper) ApplyUpdate(in UpdateAccountInput, acc *domain.Account) {
	acc.Name = in.Name
	acc.Type = domain.AccountType(in.Type)
	acc.Currency = domain.Currency(in.Currency)
	acc.ProviderGroup = in.ProviderGroup
	acc.IsInvested = in.IsInvested
}

func (mapper) ToOutput(a domain.Account) AccountOutput {
	return AccountOutput{
		ID:            a.ID,
		Name:          a.Name,
		Type:          string(a.Type),
		Currency:      string(a.Currency),
		ProviderGroup: a.ProviderGroup,
		IsInvested:    a.IsInvested,
		IsActive:      a.IsActive,
		CreatedAt:     a.CreatedAt,
		UpdatedAt:     a.UpdatedAt,
	}
}

func (m mapper) ToOutputList(accounts []domain.Account) []AccountOutput {
	out := make([]AccountOutput, len(accounts))
	for i, a := range accounts {
		out[i] = m.ToOutput(a)
	}
	return out
}
