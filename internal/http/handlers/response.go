// Package handlers contiene los handlers REST por recurso (Plan Técnico
// sección 9). Traducen HTTP <-> dominio y delegan toda la lógica en
// internal/service; no acceden a repositorios directamente.
package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/domain"
	"github.com/mauricioarielramirez/financial-tracking-app-backend/internal/repository"
)

// writeJSON serializa v como JSON con el status dado.
func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if v == nil {
		return
	}
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("error codificando respuesta JSON: %v", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

// writeError mapea errores conocidos del dominio/repositorio a un status
// HTTP razonable; cualquier otro error se trata como 500 sin filtrar
// detalles internos al cliente.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		writeJSON(w, http.StatusNotFound, errorResponse{Error: "no encontrado"})
	case errors.Is(err, repository.ErrConflict):
		writeJSON(w, http.StatusConflict, errorResponse{Error: err.Error()})
	case errors.Is(err, domain.ErrAccountNameRequired),
		errors.Is(err, domain.ErrAccountTypeInvalid),
		errors.Is(err, domain.ErrAccountCurrencyRequired):
		writeJSON(w, http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		log.Printf("error interno: %v", err)
		writeJSON(w, http.StatusInternalServerError, errorResponse{Error: "error interno"})
	}
}
