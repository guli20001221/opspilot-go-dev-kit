package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"

	"opspilot-go/internal/ingestion"
)

type ingestDocumentRequest struct {
	TenantID         string `json:"tenant_id"`
	DocumentID       string `json:"document_id"`
	DocumentVersion  int    `json:"document_version"`
	SourceTitle      string `json:"source_title"`
	SourceURI        string `json:"source_uri"`
	Content          string `json:"content"`
	PermissionsScope string `json:"permissions_scope"`
}

type ingestDocumentResponse struct {
	DocumentID   string `json:"document_id"`
	TenantID     string `json:"tenant_id"`
	ChunksStored int    `json:"chunks_stored"`
	ParentChunks int    `json:"parent_chunks"`
	ChildChunks  int    `json:"child_chunks"`
}

func (a *appHandler) handleDocuments(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}

	if a.ingestion == nil {
		writeError(w, http.StatusServiceUnavailable, "ingestion_unavailable", "ingestion pipeline not configured")
		return
	}

	var req ingestDocumentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid json body")
		return
	}
	if strings.TrimSpace(req.TenantID) == "" || strings.TrimSpace(req.DocumentID) == "" || strings.TrimSpace(req.Content) == "" {
		writeError(w, http.StatusBadRequest, "invalid_document", "tenant_id, document_id, and content are required")
		return
	}

	result, err := a.ingestion.Ingest(r.Context(), ingestion.Document{
		DocumentID:       strings.TrimSpace(req.DocumentID),
		TenantID:         strings.TrimSpace(req.TenantID),
		DocumentVersion:  req.DocumentVersion,
		SourceTitle:      req.SourceTitle,
		SourceURI:        req.SourceURI,
		Content:          req.Content,
		PermissionsScope: req.PermissionsScope,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ingestion_failed", err.Error())
		return
	}

	writeJSON(w, http.StatusCreated, ingestDocumentResponse{
		DocumentID:   result.DocumentID,
		TenantID:     result.TenantID,
		ChunksStored: result.ChunksStored,
		ParentChunks: result.ParentChunks,
		ChildChunks:  result.ChildChunks,
	})
}
