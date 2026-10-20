package account_test

// Este archivo depende de dos mocks generados con gomock que todavía no
// existen en el árbol (mapper_mock_test.go en este mismo paquete, y
// internal/repository/mocks/repository_mock.go): por eso no compila hasta
// correr, una sola vez:
//
//	go install go.uber.org/mock/mockgen@v0.6.0
//	go generate ./...
//
// Ver README para el detalle. Se deja igual como referencia de diseño: es
// el motivo por el que Mapper y AccountRepository son interfaces.

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

	// El mapper puede devolver cualquier cosa: lo que se afirma acá es que,
	// si el resultado no pasa Validate() (nombre vacío), el usecase nunca
	// debe tocar el repositorio. gomock.Times(0) hace que el test falle si
	// Create se invoca de todas formas.
	mapper.EXPECT().
		ToDomain(in).
		Return(domain.Account{Name: "", Type: domain.AccountTypeBank, Currency: domain.CurrencyARS})
	repo.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)

	uc := account.New(repo, mapper)
	if _, err := uc.Create(context.Background(), in); err == nil {
		t.Fatal("se esperaba un error de validación, no hubo ninguno")
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
			a.ID = "abc-123" // simula lo que hace el repository real
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
		acc.Name = newName // simula lo que hace el mapper real: sólo pisa Name
	})
	repo.EXPECT().ExistsByName(gomock.Any(), newName, gomock.Any()).Return(false, nil)
	repo.EXPECT().Update(gomock.Any(), existing).Return(nil)
	// ToOutput se llama después de que ApplyPatch ya mutó existing.Name, así
	// que el matcher no puede capturar *existing por valor de antemano.
	mapper.EXPECT().ToOutput(gomock.Any()).Return(expectedOutput)

	uc := account.New(repo, mapper)
	out, err := uc.Patch(context.Background(), "abc-123", in)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if out != expectedOutput {
		t.Fatalf("output inesperado: %+v", out)
	}
	// Type y Currency no vinieron en el patch: deben conservar su valor original.
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
