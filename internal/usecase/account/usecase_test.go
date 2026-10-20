package account_test

// This file depends on two gomock-generated mocks that don't exist in the
// tree yet (mapper_mock_test.go in this same package, and
// internal/repository/mocks/repository_mock.go): that's why it won't
// compile until you run, once:
//
//	go install go.uber.org/mock/mockgen@v0.6.0
//	go generate ./...
//
// See the README for details. It's kept as-is as a design reference: this
// is the reason Mapper and AccountRepository are interfaces.

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository/mocks"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/account"
)

func TestCreate_NoLLamaAlRepositorioSiElMapeoDaUnaCuentaInvalida(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccountRepository(ctrl)
	mapper := account.NewMockMapper(ctrl)

	in := account.CreateAccountInput{Name: "", Type: "bank", Currency: "ARS"}

	// The mapper can return anything: what's being asserted here is that,
	// if the result doesn't pass Validate() (empty name), the usecase must
	// never touch the repository. gomock.Times(0) makes the test fail if
	// Create gets invoked anyway.
	mapper.EXPECT().
		ToDomain(in).
		Return(domain.Account{Name: "", Type: domain.AccountTypeBank, Currency: domain.CurrencyARS})
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	uc := account.New(repo, mapper)
	if _, err := uc.Create(context.Background(), in); err == nil {
		t.Fatal("expected a validation error, got none")
	}
}

func TestCreate_PersisteYDevuelveElOutputDelMapper(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccountRepository(ctrl)
	mapper := account.NewMockMapper(ctrl)

	in := account.CreateAccountInput{Name: "Cuenta ARS", Type: "bank", Currency: "ARS"}
	domainAccount := domain.Account{Name: "Cuenta ARS", Type: domain.AccountTypeBank, Currency: domain.CurrencyARS}
	expectedOutput := account.AccountOutput{ID: "abc-123", Name: "Cuenta ARS"}

	mapper.EXPECT().ToDomain(in).Return(domainAccount)
	repo.EXPECT().ExistsByName(gomock.Any(), domainAccount.Name, (*string)(nil)).Return(false, nil)
	repo.EXPECT().Create(gomock.Any(), &domainAccount).DoAndReturn(
		func(_ context.Context, a *domain.Account) error {
			a.ID = "abc-123" // simulates what the real repository does
			return nil
		},
	)
	mapper.EXPECT().ToOutput(gomock.Any()).Return(expectedOutput)

	uc := account.New(repo, mapper)
	out, err := uc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if out != expectedOutput {
		t.Fatalf("output inesperado: %+v", out)
	}
}

func TestPatch_SoloPisaElCampoPresenteYConservaElResto(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccountRepository(ctrl)
	mapper := account.NewMockMapper(ctrl)

	existing := &domain.Account{
		ID:       "abc-123",
		Name:     "Cuenta ARS",
		Type:     domain.AccountTypeBank,
		Currency: domain.CurrencyARS,
	}
	newName := "Cuenta ARS Renombrada"
	in := account.PatchAccountInput{Name: &newName}
	expectedOutput := account.AccountOutput{ID: "abc-123", Name: newName}

	repo.EXPECT().GetByID(gomock.Any(), "abc-123").Return(existing, nil)
	mapper.EXPECT().ApplyPatch(in, existing).Do(func(_ account.PatchAccountInput, acc *domain.Account) {
		acc.Name = newName // simulates what the real mapper does: only overwrites Name
	})
	repo.EXPECT().ExistsByName(gomock.Any(), newName, gomock.Any()).Return(false, nil)
	repo.EXPECT().Update(gomock.Any(), existing).Return(nil)
	// ToOutput is called after ApplyPatch already mutated existing.Name, so
	// the matcher can't capture *existing by value ahead of time.
	mapper.EXPECT().ToOutput(gomock.Any()).Return(expectedOutput)

	uc := account.New(repo, mapper)
	out, err := uc.Patch(context.Background(), "abc-123", in)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if out != expectedOutput {
		t.Fatalf("output inesperado: %+v", out)
	}
	// Type and Currency weren't in the patch: they must keep their original value.
	if existing.Type != domain.AccountTypeBank || existing.Currency != domain.CurrencyARS {
		t.Fatalf("se modificaron campos que no vinieron en el patch: %+v", existing)
	}
}

func TestCreate_RechazaNombreDuplicadoSinLlamarACreate(t *testing.T) {
	ctrl := gomock.NewController(t)
	repo := mocks.NewMockAccountRepository(ctrl)
	mapper := account.NewMockMapper(ctrl)

	in := account.CreateAccountInput{Name: "Cuenta ARS", Type: "bank", Currency: "ARS"}
	domainAccount := domain.Account{Name: "Cuenta ARS", Type: domain.AccountTypeBank, Currency: domain.CurrencyARS}

	mapper.EXPECT().ToDomain(in).Return(domainAccount)
	repo.EXPECT().ExistsByName(gomock.Any(), domainAccount.Name, (*string)(nil)).Return(true, nil)
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	uc := account.New(repo, mapper)
	if _, err := uc.Create(context.Background(), in); err == nil {
		t.Fatal("se esperaba un error de nombre duplicado, no hubo ninguno")
	}
}
