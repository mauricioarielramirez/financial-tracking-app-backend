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

func (h *SnapshotHandler) Create(w http.ResponseWriter, r *http.Request) {
	var in snapshot.CreateSnapshotInput
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

// List implementa GET /snapshots con los filtros de RF-25: ?from=, ?to=
// (fechas en formato RFC3339, p. ej. 2022-01-31T00:00:00Z).
func (h *SnapshotHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := snapshot.ListFilter{}

	if from := r.URL.Query().Get("from"); from != "" {
		t, err := time.Parse(time.RFC3339, from)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "parámetro 'from' inválido, se espera RFC3339"})
			return
		}
		filter.From = &t
	}
	if to := r.URL.Query().Get("to"); to != "" {
		t, err := time.Parse(time.RFC3339, to)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errorResponse{Error: "parámetro 'to' inválido, se espera RFC3339"})
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

func (h *SnapshotHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	out, err := h.usecase.GetByID(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (h *SnapshotHandler) Update(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")

	var in snapshot.UpdateSnapshotInput
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
func (h *SnapshotHandler) Import(w http.ResponseWriter, r *http.Request) {
	var in snapshot.ImportInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: "cuerpo JSON inválido"})
		return
	}

	out := h.usecase.Import(r.Context(), in)
	writeJSON(w, http.StatusOK, out)
}
