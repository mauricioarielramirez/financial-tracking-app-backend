package snapshot

import (
	"context"
	"time"

	"github.com/shopspring/decimal"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
)

// UseCase orchestrates the snapshot repository and the account one: it
// needs to know which accounts are active (and their currency/invested
// classification) to be able to apply domain.BuildSnapshotDetail (RF-05 to
// RF-09, RF-13 to RF-16).
type UseCase struct {
	repo        repository.SnapshotRepository
	accountRepo repository.AccountRepository
	mapper      Mapper
}

func New(repo repository.SnapshotRepository, accountRepo repository.AccountRepository, mapper Mapper) *UseCase {
	return &UseCase{repo: repo, accountRepo: accountRepo, mapper: mapper}
}

// Create builds and persists a monthly snapshot (RF-05, RF-06), first
// checking that one doesn't already exist for the same month, unless
// Force is set (RF-08).
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

// Update replaces an already-loaded snapshot's date, notes, balances, and
// exchange rates (RF-07), reapplying the same per-month duplicate
// validation when the date moves to a different month (RF-08).
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

// Delete removes a snapshot and its associated breakdown (cascade handled
// by the repository).
func (uc *UseCase) Delete(ctx context.Context, id string) error {
	return uc.repo.Delete(ctx, id)
}

// GetByID builds the full detail of an already-persisted snapshot (RF-21,
// RF-22), reconstructing the totals from what was saved (it never
// recalculates BalanceARS with current exchange rates: RNF-02).
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

	// Every account is listed (active and inactive) because an account
	// deactivated after this snapshot must still be classified in its
	// historical breakdown (RF-04, RF-16).
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

// List returns the summarized snapshot listing filtered by date range
// (RF-25), with each one's consolidated position (RF-22).
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

// Import bulk-loads the historical snapshots from 2022 (RF-09). Each item
// is processed independently: an error in one doesn't abort the rest, and
// unlike Create/Update it doesn't require covering "all currently active
// accounts" (the account portfolio may have changed since then) nor does
// it validate per-month duplicates (an import reconstructs a complete
// historical series).
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

// accountsReferencedBy resolves, by ID, the accounts used by an import
// item (unlike Create/Update, it doesn't require them to be the currently
// active accounts).
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

// buildDetail builds the domain.Snapshot and its SnapshotDetail from an
// input shared by Create/Update/Import. isImport controls the default
// source for exchange rates without an explicit Source (RF-06, RF-09).
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
