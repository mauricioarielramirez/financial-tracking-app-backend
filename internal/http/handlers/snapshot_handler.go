package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/usecase/snapshot"
)

// SnapshotHandler expone RF-05 a RF-09 y RF-11 a RF-16 vía HTTP. Sólo
// traduce HTTP ⇄ DTO del usecase: no conoce domain.Snapshot en ningún
// momento.
type SnapshotHandler struct {
	usecase *snapshot.UseCase
}

func NewSnapshotHandler(uc *snapshot.UseCase) *SnapshotHandler {
	return &SnapshotHandler{usecase: uc}
}

// Routes registra los endpoints de snapshots bajo el router recibido (se
// monta en /api/v1/snapshots desde cmd/api/main.go).
func (h *SnapshotHandler) Routes(r chi.Router) {
	r.Post("/", h.Create)
	r.Post("/import", h.Import)
	r.Get("/", h.List)
	r.Get("/{id}", h.Get)
	r.Put("/{id}", h.Update)
	r.Delete("/{id}", h.Delete)
}

// Create implementa POST /api/v1/snapshots (RF-05, RF-06).
//
//	@Summary	Create monthly snapshot
//	@Tags		snapshots
//	@Accept		json
//	@Produce	json
//	@Param		body	body		snapshot.CreateSnapshotInput	true	"Snapshot to create"
//	@Success	201		{object}	snapshot.SnapshotOutput
//	@Failure	400		{object}	errorResponse
//	@Failure	409		{object}	errorResponse	"a snapshot already exists for that month (use force to override, RF-08)"
//	@Router		/api/v1/snapshots [post]
func (h *SnapshotHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in snapshot.CreateSnapshotInput
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

// List implementa GET /snapshots con los filtros de RF-25: ?from=, ?to=
// (fechas en formato RFC3339, p. ej. 2022-01-31T00:00:00Z).
//
//	@Summary	List snapshots
//	@Tags		snapshots
//	@Produce	json
//	@Param		from	query		string	false	"RFC3339 date, inclusive lower bound"
//	@Param		to		query		string	false	"RFC3339 date, inclusive upper bound"
//	@Success	200		{array}		snapshot.SnapshotSummary
//	@Failure	400		{object}	errorResponse
//	@Router		/api/v1/snapshots [get]
func (h *SnapshotHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := snapshot.ListFilter{}

	if from := r.URL.Query().Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid 'from' parameter, expected RFC3339"})
			return
		}
		filter.From = &t
	}
	if to := r.URL.Query().Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid 'to' parameter, expected RFC3339"})
			return
		}
		filter.To = &t
	}

	out, err := h.usecase.List(r.Context(), filter)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Get implementa GET /api/v1/snapshots/{id}.
//
//	@Summary	Get snapshot by id
//	@Tags		snapshots
//	@Produce	json
//	@Param		id	path		string	true	"Snapshot ID"
//	@Success	200	{object}	snapshot.SnapshotOutput
//	@Failure	404	{object}	errorResponse
//	@Router		/api/v1/snapshots/{id} [get]
func (h *SnapshotHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

// Update implementa PUT /api/v1/snapshots/{id} (RF-07): reemplazo completo.
//
//	@Summary	Replace snapshot (full update)
//	@Tags		snapshots
//	@Accept		json
//	@Produce	json
//	@Param		id		path		string							true	"Snapshot ID"
//	@Param		body	body		snapshot.UpdateSnapshotInput	true	"Full snapshot payload"
//	@Success	200		{object}	snapshot.SnapshotOutput
//	@Failure	400		{object}	errorResponse
//	@Failure	404		{object}	errorResponse
//	@Failure	409		{object}	errorResponse	"a snapshot already exists for the new month (use force, RF-08)"
//	@Router		/api/v1/snapshots/{id} [put]
func (h *SnapshotHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in snapshot.UpdateSnapshotInput
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

// Delete implementa DELETE /api/v1/snapshots/{id}.
//
//	@Summary	Delete snapshot
//	@Tags		snapshots
//	@Param		id	path	string	true	"Snapshot ID"
//	@Success	204
//	@Failure	404	{object}	errorResponse
//	@Router		/api/v1/snapshots/{id} [delete]
func (h *SnapshotHandler) Delete(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.usecase.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusNoContent, nil)
}

// Import implementa POST /snapshots/import (RF-09): carga masiva de los
// snapshots históricos de 2022. Siempre responde 200 con el detalle
// por-ítem: un ítem fallido no es un error de la request completa.
//
//	@Summary	Bulk import historical snapshots
//	@Tags		snapshots
//	@Accept		json
//	@Produce	json
//	@Param		body	body		snapshot.ImportInput	true	"Snapshots to import"
//	@Success	200		{object}	snapshot.ImportOutput
//	@Failure	400		{object}	errorResponse
//	@Router		/api/v1/snapshots/import [post]
func (h *SnapshotHandler) Import(w http.ResponseWriter, r *http.Request) {
	var in snapshot.ImportInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "invalid JSON body"})
		return
	}

	out := h.usecase.Import(r.Context(), in)
	writeJSON(w, http.StatusOK, out)
}
