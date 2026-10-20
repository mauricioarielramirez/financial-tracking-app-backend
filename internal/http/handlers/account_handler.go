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
	r.Patch("/{id}", h.Patch)
	r.Delete("/{id}", h.Deactivate)
}

// Create implementa POST /api/v1/accounts (RF-01).
//
//	@Summary	Create account
//	@Tags		accounts
//	@Accept		json
//	@Produce	json
//	@Param		body	body		account.CreateAccountInput	true	"Account to create"
//	@Success	201		{object}	account.AccountOutput
//	@Failure	400		{object}	errorResponse
//	@Failure	409		{object}	errorResponse	"an account with that name already exists"
//	@Router		/api/v1/accounts [post]
func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in account.CreateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
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
//
//	@Summary	List accounts
//	@Tags		accounts
//	@Produce	json
//	@Param		active	query		bool	false	"only active accounts"
//	@Param		type	query		string	false	"filter by account type"
//	@Param		group	query		string	false	"filter by provider group"
//	@Success	200		{array}		account.AccountOutput
//	@Router		/api/v1/accounts [get]
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

// Get implementa GET /api/v1/accounts/{id}.
//
//	@Summary	Get account by id
//	@Tags		accounts
//	@Produce	json
//	@Param		id	path		string	true	"Account ID"
//	@Success	200	{object}	account.AccountOutput
//	@Failure	404	{object}	errorResponse
//	@Router		/api/v1/accounts/{id} [get]
func (h *AccountHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Update implementa PUT /api/v1/accounts/{id}: reemplazo completo, hay que
// mandar todos los campos editables (no es un merge parcial, ver Patch).
//
//	@Summary	Replace account (full update)
//	@Tags		accounts
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Account ID"
//	@Param		body	body		account.UpdateAccountInput	true	"Full account payload"
//	@Success	200		{object}	account.AccountOutput
//	@Failure	400		{object}	errorResponse
//	@Failure	404		{object}	errorResponse
//	@Failure	409		{object}	errorResponse	"an account with that name already exists"
//	@Router		/api/v1/accounts/{id} [put]
func (h *AccountHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in account.UpdateAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	out, err := h.usecase.Update(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Patch implementa PATCH /accounts/{id}: edición parcial, sólo se pisan los
// campos presentes en el JSON (ver account.PatchAccountInput).
//
//	@Summary	Partially update account
//	@Tags		accounts
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string						true	"Account ID"
//	@Param		body	body		account.PatchAccountInput	true	"Only the fields to change"
//	@Success	200		{object}	account.AccountOutput
//	@Failure	400		{object}	errorResponse
//	@Failure	404		{object}	errorResponse
//	@Failure	409		{object}	errorResponse	"an account with that name already exists"
//	@Router		/api/v1/accounts/{id} [patch]
func (h *AccountHandler) Patch(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in account.PatchAccountInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	out, err := h.usecase.Patch(r.Context(), id, in)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Deactivate implementa el DELETE de RF-01: soft delete, nunca borra el
// historial (RF-04).
//
//	@Summary	Deactivate account (soft delete)
//	@Tags		accounts
//	@Param		id	path	string	true	"Account ID"
//	@Success	204
//	@Failure	404	{object}	errorResponse
//	@Router		/api/v1/accounts/{id} [delete]
func (h *AccountHandler) Deactivate(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.Deactivate(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}
