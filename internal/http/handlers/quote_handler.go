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

func (h *QuoteHandler) GetSuggested(w http.ResponseWriter, r *http.Request) {
	out := h.usecase.GetSuggested(r.Context())
	writeJSON(w, http.StatusOK, out)
}
