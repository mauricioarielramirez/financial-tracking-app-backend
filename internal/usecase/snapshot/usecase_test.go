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

// TestCreate_RechazaDuplicadoDeMesSinForce covers RF-08: if a snapshot
// already exists for the same month and Force isn't requested, the
// usecase must return the conflict without touching the account
// repository or the snapshot one beyond the check.
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

// TestCreate_PersisteYDevuelveElOutputDelMapper covers the happy path of
// RF-05/RF-06: with no month duplicate, it builds the detail from the
// active accounts and persists it.
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
	// BalancesByAccount and ToDomainRates are called with the empty
	// inputs; we also return empty so domain.BuildSnapshotDetail doesn't
	// find any account to validate.
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

// TestCreate_ForcePermiteOmitirLaVerificacionDeMes covers RF-08's
// override: with Force set to true, the usecase must not query
// ExistsForMonth.
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
