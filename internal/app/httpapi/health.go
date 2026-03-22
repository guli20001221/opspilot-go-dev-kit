package httpapi

import (
	"encoding/json"
	"net/http"

	toolregistry "opspilot-go/internal/tools/registry"
	"opspilot-go/internal/workflow"
)

type statusResponse struct {
	Status string `json:"status"`
}

// Dependencies supplies optional runtime services for the HTTP layer.
type Dependencies struct {
	Workflows *workflow.Service
	Registry  *toolregistry.Registry
}

// NewHandler constructs the minimum API handler tree for the foundation slice.
func NewHandler() http.Handler {
	return NewHandlerWithDependencies(Dependencies{})
}

// NewHandlerWithDependencies constructs the HTTP handler tree with injected services.
func NewHandlerWithDependencies(deps Dependencies) http.Handler {
	mux := http.NewServeMux()
	app := newAppHandler(deps.Workflows, deps.Registry)
	mux.HandleFunc("/healthz", writeStatus("ok"))
	mux.HandleFunc("/readyz", writeStatus("ready"))
	app.registerRoutes(mux)

	return withRequestContext(mux)
}

func writeStatus(status string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		_ = json.NewEncoder(w).Encode(statusResponse{Status: status})
	}
}
