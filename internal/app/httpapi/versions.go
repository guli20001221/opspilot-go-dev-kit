package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"opspilot-go/internal/version"
)

type versionResponse struct {
	VersionID           string `json:"version_id"`
	RuntimeVersion      string `json:"runtime_version"`
	Provider            string `json:"provider,omitempty"`
	Model               string `json:"model,omitempty"`
	PromptBundle        string `json:"prompt_bundle"`
	PlannerVersion      string `json:"planner_version"`
	RetrievalVersion    string `json:"retrieval_version"`
	ToolRegistryVersion string `json:"tool_registry_version"`
	CriticVersion       string `json:"critic_version"`
	WorkflowVersion     string `json:"workflow_version"`
	Notes               string `json:"notes,omitempty"`
	CreatedAt           string `json:"created_at"`
}

type listVersionsResponse struct {
	Versions   []versionResponse `json:"versions"`
	HasMore    bool              `json:"has_more"`
	NextOffset *int              `json:"next_offset,omitempty"`
}

func (a *appHandler) handleVersions(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	filter, err := parseVersionListFilter(r)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_query", err.Error())
		return
	}

	page, err := a.versions.ListVersions(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "version_list_failed", err.Error())
		return
	}

	resp := listVersionsResponse{
		Versions: make([]versionResponse, 0, len(page.Versions)),
		HasMore:  page.HasMore,
	}
	if page.HasMore {
		resp.NextOffset = &page.NextOffset
	}
	for _, item := range page.Versions {
		resp.Versions = append(resp.Versions, newVersionResponse(item))
	}

	writeJSON(w, http.StatusOK, resp)
}

func (a *appHandler) handleVersionByID(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	versionID := strings.TrimPrefix(r.URL.Path, "/api/v1/versions/")
	if versionID == "" || strings.Contains(versionID, "/") {
		writeError(w, http.StatusNotFound, "not_found", "not found")
		return
	}

	item, err := a.versions.GetVersion(r.Context(), versionID)
	if err != nil {
		if errors.Is(err, version.ErrVersionNotFound) {
			writeError(w, http.StatusNotFound, "version_not_found", "version not found")
			return
		}

		writeError(w, http.StatusInternalServerError, "version_lookup_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusOK, newVersionResponse(item))
}

func parseVersionListFilter(r *http.Request) (version.ListFilter, error) {
	filter := version.ListFilter{Limit: 20}
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		limit, err := strconv.Atoi(rawLimit)
		if err != nil || limit <= 0 {
			return version.ListFilter{}, errors.New("limit must be a positive integer")
		}
		filter.Limit = limit
	}
	if rawOffset := r.URL.Query().Get("offset"); rawOffset != "" {
		offset, err := strconv.Atoi(rawOffset)
		if err != nil || offset < 0 {
			return version.ListFilter{}, errors.New("offset must be a non-negative integer")
		}
		filter.Offset = offset
	}

	return filter, nil
}

func newVersionResponse(item version.Version) versionResponse {
	return versionResponse{
		VersionID:           item.ID,
		RuntimeVersion:      item.RuntimeVersion,
		Provider:            item.Provider,
		Model:               item.Model,
		PromptBundle:        item.PromptBundle,
		PlannerVersion:      item.PlannerVersion,
		RetrievalVersion:    item.RetrievalVersion,
		ToolRegistryVersion: item.ToolRegistryVersion,
		CriticVersion:       item.CriticVersion,
		WorkflowVersion:     item.WorkflowVersion,
		Notes:               item.Notes,
		CreatedAt:           item.CreatedAt.Format(time.RFC3339Nano),
	}
}
