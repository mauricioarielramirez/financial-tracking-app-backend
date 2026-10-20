package account

import "github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"

//go:generate mockgen -source=mapper.go -destination=mapper_mock_test.go -package=account

// Mapper translates domain.Account ⇄ this package's DTOs. It's an
// interface (not loose functions) so UseCase can be tested with a
// fake/mocked mapper without running the real mapping logic, and so the
// real mapper can be tested on its own.
type Mapper interface {
	// ToDomain builds a new account from the create input (RF-01). It
	// doesn't set ID/IsActive/CreatedAt/UpdatedAt: the repository fills
	// those in on persist.
	ToDomain(in CreateAccountInput) domain.Account

	// ApplyUpdate applies UpdateAccountInput's editable fields onto an
	// already-existing account, without touching ID, IsActive, or the
	// timestamps.
	ApplyUpdate(in UpdateAccountInput, acc *domain.Account)

	// ApplyPatch applies only the present (non-nil) fields of
	// PatchAccountInput onto an already-existing account; absent fields
	// keep their current value.
	ApplyPatch(in PatchAccountInput, acc *domain.Account)

	ToOutput(a domain.Account) AccountOutput
	ToOutputList(accounts []domain.Account) []AccountOutput
}

// mapper is Mapper's only real implementation. It has no state or
// dependencies, so a zero value (mapper{}) is enough.
type mapper struct{}

// NewMapper builds the real account mapper.
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

func (mapper) ApplyPatch(in PatchAccountInput, acc *domain.Account) {
	if in.Name != nil {
		acc.Name = *in.Name
	}
	if in.Type != nil {
		acc.Type = domain.AccountType(*in.Type)
	}
	if in.Currency != nil {
		acc.Currency = domain.Currency(*in.Currency)
	}
	if in.ProviderGroup != nil {
		acc.ProviderGroup = in.ProviderGroup
	}
	if in.IsInvested != nil {
		acc.IsInvested = *in.IsInvested
	}
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
