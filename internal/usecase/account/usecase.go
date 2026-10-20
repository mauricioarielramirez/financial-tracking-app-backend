package account

import (
	"context"
	"fmt"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
)

// ErrDuplicateAccountName is returned when another account already exists
// with the same name (on create or edit).
var ErrDuplicateAccountName = fmt.Errorf("%w: an account with that name already exists", repository.ErrConflict)

// UseCase orchestrates the account repository (RF-01 to RF-04). This
// aggregate doesn't need a pure-calculation service: the only one that
// shows up later is snapshots' (% distribution, consolidated position).
type UseCase struct {
	repo   repository.AccountRepository
	mapper Mapper
}

// New builds the usecase. repo and mapper are interfaces: in tests they're
// swapped for mocks generated with gomock (see mapper.go and
// internal/repository/repository.go).
func New(repo repository.AccountRepository, mapper Mapper) *UseCase {
	return &UseCase{repo: repo, mapper: mapper}
}

func (uc *UseCase) Create(ctx context.Context, in CreateAccountInput) (AccountOutput, error) {
	acc := uc.mapper.ToDomain(in)
	if err := acc.Validate(); err != nil {
		return AccountOutput{}, err
	}

	exists, err := uc.repo.ExistsByName(ctx, acc.Name, nil)
	if err != nil {
		return AccountOutput{}, err
	}
	if exists {
		return AccountOutput{}, ErrDuplicateAccountName
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
	return uc.validateAndSave(ctx, id, acc)
}

// Patch implements PATCH /accounts/{id}: unlike Update (a full replace),
// it only overwrites the fields present in the JSON (see
// PatchAccountInput); absent ones keep their current value.
func (uc *UseCase) Patch(ctx context.Context, id string, in PatchAccountInput) (AccountOutput, error) {
	acc, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return AccountOutput{}, err
	}

	uc.mapper.ApplyPatch(in, acc)
	return uc.validateAndSave(ctx, id, acc)
}

// validateAndSave applies the domain invariants, checks that the name
// (already normalized by acc.Validate()) doesn't collide with another
// account, and persists. Shared by Update and Patch: both only differ in
// which fields they overwrite before getting here.
func (uc *UseCase) validateAndSave(ctx context.Context, id string, acc *domain.Account) (AccountOutput, error) {
	if err := acc.Validate(); err != nil {
		return AccountOutput{}, err
	}

	exists, err := uc.repo.ExistsByName(ctx, acc.Name, &id)
	if err != nil {
		return AccountOutput{}, err
	}
	if exists {
		return AccountOutput{}, ErrDuplicateAccountName
	}

	if err := uc.repo.Update(ctx, acc); err != nil {
		return AccountOutput{}, err
	}
	return uc.mapper.ToOutput(*acc), nil
}

// Deactivate does a soft delete (RF-04): this account's already-loaded
// balance history is kept unchanged (the repository guarantees this).
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
