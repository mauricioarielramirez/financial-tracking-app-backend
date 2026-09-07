package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/account"
)

// AccountHandler expone RF-01 a RF-04 vía HTTP. Sólo traduce HTTP ⇄ DTO del
// usecase (account.CreateAccountInput, account.AccountOutput, etc.): no
// conoce domain.Account en ningún momento.
type AccountHandler struct {
	usecase *account.UseCase
}

func NewAccountHandler(uc *account.UseCase) *AccountHandler {
	return &AccountHandler{usecase: uc}
}

// Routes registra los endpoints de cuentas bajo el router recibido
// (se monta en /api/v1/accounts desde cmd/api/main.go).
func (h *AccountHandler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Deactivate)
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in account.CreateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cuerpo JSON inválido"})
		return
	}

	out, err := h.usecase.Create(r.Context(), in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, out)
}

// List implementa GET /accounts con los filtros de RF-25:
// ?active=true, ?type=, ?group=
func (h *AccountHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := account.ListFilter{}

	if active := r.URL.Query().Get("active"); active == "true" {
		v := true
		filter.ActiveOnly = &v
	}
	if t := r.URL.Query().Get("type"); t != "" {
		filter.Type = &t
	}
	if g := r.URL.Query().Get("group"); g != "" {
		filter.Group = &g
	}

	out, err := h.usecase.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in account.UpdateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cuerpo JSON inválido"})
		return
	}

	out, err := h.usecase.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Deactivate implementa el DELETE de RF-01: soft delete, nunca borra el
// historial (RF-04).
func (h *AccountHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.Deactivate(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
