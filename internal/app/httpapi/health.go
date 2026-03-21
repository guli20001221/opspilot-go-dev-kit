package httpapi

import (
	"encoding/json"
	"net/http"
)

type statusResponse struct {
	Status string `json:"status"`
}

// NewHandler constructs the minimum API handler tree for the foundation slice.
func NewHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", writeStatus("ok"))
	mux.HandleFunc("/readyz", writeStatus("ready"))

	return withRequestContext(mux)
}

func writeStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(statusResponse{Status: status})
	}
}
