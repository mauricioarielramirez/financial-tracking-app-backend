package snapshot_test

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository/mocks"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/snapshot"
)

// TestCreate_RechazaDuplicadoDeMesSinForce cubre RF-08: si ya existe un
// snapshot en el mismo mes y no se pide Force, el usecase debe devolver el
// conflicto sin tocar el repositorio de cuentas ni el de snapshots más allá
// de la verificación.
func TestCreate_RechazaDuplicadoDeMesSinForce(t *testing.T) {
	ctrl := gomock.NewController(t)
	snapRepo := mocks.NewMockSnapshotRepository(ctrl)
	accountRepo := mocks.NewMockAccountRepository(ctrl)
	mapper := snapshot.NewMockMapper(ctrl)

	in := snapshot.CreateSnapshotInput{}

	snapRepo.EXPECT().ExistsForMonth(gomock.Any(), in.SnapshotDate).Return(true, nil)
	accountRepo.EXPECT().List(gomock.Any(), gomock.Any()).Times(0)
	snapRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	uc := snapshot.New(snapRepo, accountRepo, mapper)
	if _, err := uc.Create(context.Background(), in); err != repository.ErrConflict {
		t.Fatalf("se esperaba repository.ErrConflict, se obtuvo: %v", err)
	}
}

// TestCreate_PersisteYDevuelveElOutputDelMapper cubre el camino feliz de
// RF-05/RF-06: sin duplicado de mes, arma el detalle a partir de las cuentas
// activas y persiste.
func TestCreate_PersisteYDevuelveElOutputDelMapper(t *testing.T) {
	ctrl := gomock.NewController(t)
	snapRepo := mocks.NewMockSnapshotRepository(ctrl)
	accountRepo := mocks.NewMockAccountRepository(ctrl)
	mapper := snapshot.NewMockMapper(ctrl)

	in := snapshot.CreateSnapshotInput{
		Balances: []snapshot.BalanceInput{},
		Rates:    []snapshot.RateInput{},
	}
	accounts := []domain.Account{}
	expectedOutput := snapshot.SnapshotOutput{ID: "snap-123"}

	snapRepo.EXPECT().ExistsForMonth(gomock.Any(), in.SnapshotDate).Return(false, nil)
	accountRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return(accounts, nil)
	// BalancesByAccount y ToDomainRates se llaman con las entradas vacías;
	// devolvemos también vacío para que domain.BuildSnapshotDetail no
	// encuentre ninguna cuenta que validar.
	mapper.EXPECT().BalancesByAccount(in.Balances).Return(nil)
	mapper.EXPECT().ToDomainRates(in.Rates).Return(nil)
	snapRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
		func(_ context.Context, snap *domain.Snapshot, _ []domain.AccountBalance, _ []domain.ExchangeRate) error {
			snap.ID = "snap-123"
			return nil
		},
	)
	mapper.EXPECT().ToOutput(gomock.Any()).Return(expectedOutput)

	uc := snapshot.New(snapRepo, accountRepo, mapper)
	out, err := uc.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
	if out.ID != expectedOutput.ID {
		t.Fatalf("output inesperado: %+v", out)
	}
}

// TestCreate_ForcePermiteOmitirLaVerificacionDeMes cubre el override de
// RF-08: con Force en true, el usecase no debe consultar ExistsForMonth.
func TestCreate_ForcePermiteOmitirLaVerificacionDeMes(t *testing.T) {
	ctrl := gomock.NewController(t)
	snapRepo := mocks.NewMockSnapshotRepository(ctrl)
	accountRepo := mocks.NewMockAccountRepository(ctrl)
	mapper := snapshot.NewMockMapper(ctrl)

	in := snapshot.CreateSnapshotInput{Force: true}

	snapRepo.EXPECT().ExistsForMonth(gomock.Any(), gomock.Any()).Times(0)
	accountRepo.EXPECT().List(gomock.Any(), gomock.Any()).Return([]domain.Account{}, nil)
	mapper.EXPECT().BalancesByAccount(in.Balances).Return(nil)
	mapper.EXPECT().ToDomainRates(in.Rates).Return(nil)
	snapRepo.EXPECT().Create(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
	mapper.EXPECT().ToOutput(gomock.Any()).Return(snapshot.SnapshotOutput{})

	uc := snapshot.New(snapRepo, accountRepo, mapper)
	if _, err := uc.Create(context.Background(), in); err != nil {
		t.Fatalf("no se esperaba error: %v", err)
	}
}
