package account

import (
	"context"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
)

// UseCase orquesta el repository de cuentas (RF-01 a RF-04). Para este
// agregado no hace falta un service de cálculo puro: el único que aparece
// más adelante es el de snapshots (distribución %, posición consolidada).
type UseCase struct {
	repo   repository.AccountRepository
	mapper Mapper
}

// New construye el usecase. repo y mapper son interfaces: en los tests se
// reemplazan por mocks generados con gomock (ver mapper.go y
// internal/repository/repository.go).
func New(repo repository.AccountRepository, mapper Mapper) *UseCase {
	return &UseCase{repo: repo, mapper: mapper}
}

func (uc *UseCase) Create(ctx context.Context, in CreateAccountInput) (AccountOutput, error) {
	acc := uc.mapper.ToDomain(in)
	if err := acc.Validate(); err != nil {
		return AccountOutput{}, err
	}
	if err := uc.repo.Create(ctx, &acc); err != nil {
		return AccountOutput{}, err
	}
	return uc.mapper.ToOutput(acc), nil
}

func (uc *UseCase) Update(ctx context.Context, id string, in UpdateAccountInput) (AccountOutput, error) {
	acc, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return AccountOutput{}, err
	}

	uc.mapper.ApplyUpdate(in, acc)
	if err := acc.Validate(); err != nil {
		return AccountOutput{}, err
	}
	if err := uc.repo.Update(ctx, acc); err != nil {
		return AccountOutput{}, err
	}
	return uc.mapper.ToOutput(*acc), nil
}

// Deactivate hace soft delete (RF-04): el histórico de saldos ya cargados
// para esta cuenta se conserva sin cambios (lo garantiza el repository).
func (uc *UseCase) Deactivate(ctx context.Context, id string) error {
	return uc.repo.Deactivate(ctx, id)
}

func (uc *UseCase) GetByID(ctx context.Context, id string) (AccountOutput, error) {
	acc, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return AccountOutput{}, err
	}
	return uc.mapper.ToOutput(*acc), nil
}

func (uc *UseCase) List(ctx context.Context, filter ListFilter) ([]AccountOutput, error) {
	repoFilter := repository.AccountFilter{
		ActiveOnly: filter.ActiveOnly,
		Group:      filter.Group,
	}
	if filter.Type != nil {
		t := domain.AccountType(*filter.Type)
		repoFilter.Type = &t
	}

	accounts, err := uc.repo.List(ctx, repoFilter)
	if err != nil {
		return nil, err
	}
	return uc.mapper.ToOutputList(accounts), nil
}
