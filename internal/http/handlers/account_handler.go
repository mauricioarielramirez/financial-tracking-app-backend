package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/account"
)

// AccountHandler exposes RF-01 to RF-04 over HTTP. It only translates
// HTTP ⇄ usecase DTO (account.CreateAccountInput, account.AccountOutput,
// etc.): it never knows about domain.Account.
type AccountHandler struct {
	usecase *account.UseCase
}

func NewAccountHandler(uc *account.UseCase) *AccountHandler {
	return &AccountHandler{usecase: uc}
}

// Routes registers the account endpoints under the given router
// (mounted at /api/v1/accounts from cmd/api/main.go).
func (h *AccountHandler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Patch("/{id}", h.Patch)
	r.Delete("/{id}", h.Deactivate)
}

// Create implements POST /api/v1/accounts (RF-01).
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

// List implements GET /accounts with the RF-25 filters:
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

// Get implements GET /api/v1/accounts/{id}.
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

// Update implements PUT /api/v1/accounts/{id}: a full replace, you need to
// send every editable field (it's not a partial merge, see Patch).
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

// Patch implements PATCH /accounts/{id}: a partial edit, only the fields
// present in the JSON are overwritten (see account.PatchAccountInput).
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

// Deactivate implements RF-01's DELETE: a soft delete, it never erases the
// history (RF-04).
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
