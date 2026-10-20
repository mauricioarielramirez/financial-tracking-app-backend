package handlers

import (
	"net/http"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/quote"
)

// QuoteHandler expone GET /quotes/suggested (RF-10).
type QuoteHandler struct {
	usecase *quote.UseCase
}

func NewQuoteHandler(uc *quote.UseCase) *QuoteHandler {
	return &QuoteHandler{usecase: uc}
}

// GetSuggested implementa GET /api/v1/quotes/suggested (RF-10, RF-11).
//
//	@Summary	Suggested exchange rates
//	@Tags		quotes
//	@Produce	json
//	@Success	200	{object}	quote.SuggestedOutput
//	@Router		/api/v1/quotes/suggested [get]
func (h *QuoteHandler) GetSuggested(w http.ResponseWriter, r *http.Request) {
	out := h.usecase.GetSuggested(r.Context())
	writeJSON(w, http.StatusOK, out)
}
