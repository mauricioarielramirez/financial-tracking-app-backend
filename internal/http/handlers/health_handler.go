package handlers

import "net/http"

// Health es un endpoint mínimo de diagnóstico (GET /healthz), útil para
// verificar que el proceso y la conexión a la base están arriba, y como
// primer artefacto para probar el server manualmente.
func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
