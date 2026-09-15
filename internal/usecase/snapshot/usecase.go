package snapshot

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
)

// UseCase orquesta el repository de snapshots y el de cuentas: necesita
// saber qué cuentas están activas (y su moneda/clasificación invertido) para
// poder aplicar domain.BuildSnapshotDetail (RF-05 a RF-09, RF-13 a RF-16).
type UseCase struct {
	repo        repository.SnapshotRepository
	accountRepo repository.AccountRepository
	mapper      Mapper
}

func New(repo repository.SnapshotRepository, accountRepo repository.AccountRepository, mapper Mapper) *UseCase {
	return &UseCase{repo: repo, accountRepo: accountRepo, mapper: mapper}
}

// Create arma y persiste un snapshot mensual (RF-05, RF-06) verificando
// primero que no exista ya uno para el mismo mes, salvo Force (RF-08).
func (uc *UseCase) Create(ctx context.Context, in CreateSnapshotInput) (SnapshotOutput, error) {
	if !in.Force {
		exists, err := uc.repo.ExistsForMonth(ctx, in.SnapshotDate)
		if err != nil {
			return SnapshotOutput{}, err
		}
		if exists {
			return SnapshotOutput{}, repository.ErrConflict
		}
	}

	trueVal := true
	accounts, err := uc.accountRepo.List(ctx, repository.AccountFilter{ActiveOnly: &trueVal})
	if err != nil {
		return SnapshotOutput{}, err
	}

	detail, err := uc.buildDetail(in.SnapshotDate, in.Notes, accounts, in.Balances, in.Rates, false)
	if err != nil {
		return SnapshotOutput{}, err
	}

	if err := uc.repo.Create(ctx, &detail.Snapshot, detail.Balances, detail.Rates); err != nil {
		return SnapshotOutput{}, err
	}
	return uc.mapper.ToOutput(detail), nil
}

// Update reemplaza fecha, notas, saldos y cotizaciones de un snapshot ya
// cargado (RF-07), reaplicando la misma validación de duplicado por mes
// cuando la fecha cambia de mes (RF-08).
func (uc *UseCase) Update(ctx context.Context, id string, in UpdateSnapshotInput) (SnapshotOutput, error) {
	existing, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return SnapshotOutput{}, err
	}

	if !in.Force && !sameMonth(existing.SnapshotDate, in.SnapshotDate) {
		exists, err := uc.repo.ExistsForMonth(ctx, in.SnapshotDate)
		if err != nil {
			return SnapshotOutput{}, err
		}
		if exists {
			return SnapshotOutput{}, repository.ErrConflict
		}
	}

	trueVal := true
	accounts, err := uc.accountRepo.List(ctx, repository.AccountFilter{ActiveOnly: &trueVal})
	if err != nil {
		return SnapshotOutput{}, err
	}

	detail, err := uc.buildDetail(in.SnapshotDate, in.Notes, accounts, in.Balances, in.Rates, false)
	if err != nil {
		return SnapshotOutput{}, err
	}
	detail.Snapshot.ID = existing.ID
	detail.Snapshot.CreatedAt = existing.CreatedAt

	if err := uc.repo.Update(ctx, &detail.Snapshot, detail.Balances, detail.Rates); err != nil {
		return SnapshotOutput{}, err
	}
	return uc.mapper.ToOutput(detail), nil
}

// Delete elimina un snapshot y su desglose asociado (cascada a cargo del
// repository).
func (uc *UseCase) Delete(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

// GetByID arma el detalle completo de un snapshot ya persistido (RF-21,
// RF-22), reconstruyendo los totales a partir de lo guardado (nunca
// recalcula BalanceARS con cotizaciones actuales: RNF-02).
func (uc *UseCase) GetByID(ctx context.Context, id string) (SnapshotOutput, error) {
	snap, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return SnapshotOutput{}, err
	}
	balances, err := uc.repo.GetBalances(ctx, id)
	if err != nil {
		return SnapshotOutput{}, err
	}
	rates, err := uc.repo.GetRates(ctx, id)
	if err != nil {
		return SnapshotOutput{}, err
	}

	// Se listan todas las cuentas (activas e inactivas) porque una cuenta
	// desactivada después de este snapshot debe seguir clasificándose en su
	// desglose histórico (RF-04, RF-16).
	accounts, err := uc.accountRepo.List(ctx, repository.AccountFilter{})
	if err != nil {
		return SnapshotOutput{}, err
	}
	investedByAccount := make(map[string]bool, len(accounts))
	for _, a := range accounts {
		investedByAccount[a.ID] = a.IsInvested
	}

	consolidatedARS, consolidatedUSD, investedARS, notInvestedARS := domain.SummarizeStoredBalances(balances, rates, investedByAccount)

	detail := domain.SnapshotDetail{
		Snapshot:            *snap,
		Balances:            balances,
		Rates:               rates,
		ConsolidatedARS:     consolidatedARS,
		ConsolidatedUSD:     consolidatedUSD,
		InvestedTotalARS:    investedARS,
		NotInvestedTotalARS: notInvestedARS,
	}
	return uc.mapper.ToOutput(detail), nil
}

// List devuelve el listado resumido de snapshots filtrado por rango de
// fechas (RF-25), con la posición consolidada de cada uno (RF-22).
func (uc *UseCase) List(ctx context.Context, filter ListFilter) ([]SnapshotSummary, error) {
	snaps, err := uc.repo.List(ctx, repository.SnapshotFilter{From: filter.From, To: filter.To})
	if err != nil {
		return nil, err
	}

	out := make([]SnapshotSummary, len(snaps))
	for i, s := range snaps {
		balances, err := uc.repo.GetBalances(ctx, s.ID)
		if err != nil {
			return nil, err
		}
		consolidated := decimal.Zero
		for _, b := range balances {
			consolidated = consolidated.Add(b.BalanceARS)
		}
		out[i] = uc.mapper.ToSummary(s, consolidated)
	}
	return out, nil
}

// Import carga en forma masiva los snapshots históricos de 2022 (RF-09). Cada
// item se procesa de forma independiente: un error en uno no aborta el
// resto, y a diferencia de Create/Update no exige cubrir "todas las cuentas
// activas actuales" (la cartera de cuentas puede haber cambiado desde
// entonces) ni valida duplicado por mes (una importación reconstruye una
// serie histórica completa).
func (uc *UseCase) Import(ctx context.Context, in ImportInput) ImportOutput {
	results := make([]ImportItemResult, len(in.Items))

	for i, item := range in.Items {
		result := ImportItemResult{Index: i, SnapshotDate: item.SnapshotDate.Format(time.RFC3339)}

		accounts, err := uc.accountsReferencedBy(ctx, item.Balances)
		if err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		detail, err := uc.buildDetail(item.SnapshotDate, item.Notes, accounts, item.Balances, item.Rates, true)
		if err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		if err := uc.repo.Create(ctx, &detail.Snapshot, detail.Balances, detail.Rates); err != nil {
			result.Error = err.Error()
			results[i] = result
			continue
		}

		id := detail.Snapshot.ID
		result.SnapshotID = &id
		results[i] = result
	}

	return ImportOutput{Results: results}
}

// accountsReferencedBy resuelve, por ID, las cuentas usadas por un ítem de
// importación (a diferencia de Create/Update, no exige que sean las cuentas
// activas actuales).
func (uc *UseCase) accountsReferencedBy(ctx context.Context, balances []BalanceInput) ([]domain.Account, error) {
	accounts := make([]domain.Account, 0, len(balances))
	for _, b := range balances {
		acc, err := uc.accountRepo.GetByID(ctx, b.AccountID)
		if err != nil {
			return nil, err
		}
		accounts = append(accounts, *acc)
	}
	return accounts, nil
}

// buildDetail arma el domain.Snapshot y su SnapshotDetail a partir de un
// input común a Create/Update/Import. isImport controla la fuente por
// defecto de las cotizaciones sin Source explícito (RF-06, RF-09).
func (uc *UseCase) buildDetail(snapshotDate time.Time, notes *string, accounts []domain.Account, balances []BalanceInput, rateInputs []RateInput, isImport bool) (domain.SnapshotDetail, error) {
	balancesMap := uc.mapper.BalancesByAccount(balances)
	rates := uc.mapper.ToDomainRates(rateInputs)

	now := time.Now().UTC()
	defaultSource := domain.QuoteSourceManual
	if isImport {
		defaultSource = domain.QuoteSourceImportSheet
	}
	for i := range rates {
		rates[i].FetchedAt = now
		if rateInputs[i].Source == "" {
			rates[i].Source = defaultSource
		}
	}

	snap := domain.Snapshot{SnapshotDate: snapshotDate, Notes: notes}
	return domain.BuildSnapshotDetail(snap, accounts, balancesMap, rates)
}

func sameMonth(a, b time.Time) bool {
	ay, am, _ := a.Date()
	by, bm, _ := b.Date()
	return ay == by && am == bm
}
