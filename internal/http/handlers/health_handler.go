package handlers

import "net/http"

// Health is a minimal diagnostic endpoint (GET /healthz), useful to check
// that the process and the database connection are up, and as the first
// artifact to test the server by hand.
//
//	@Summary	Health check
//	@Tags		health
//	@Produce	json
//	@Success	200	{object}	map[string]string
//	@Router		/healthz [get]
func Health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}
