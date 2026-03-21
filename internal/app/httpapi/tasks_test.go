package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCreateTaskEndpoint(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","session_id":"session-1","task_type":"report_generation","reason":"workflow_required"}`)
	resp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusCreated)
	}

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.TaskID == "" {
		t.Fatal("task_id is empty")
	}
	if got.Status != "queued" {
		t.Fatalf("status = %q, want %q", got.Status, "queued")
	}
}

func TestGetTaskEndpoint(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	createBody := bytes.NewBufferString(`{"tenant_id":"tenant-1","session_id":"session-1","task_type":"report_generation","reason":"workflow_required"}`)
	createResp, err := http.Post(server.URL+"/api/v1/tasks", "application/json", createBody)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer createResp.Body.Close()

	var created taskResponse
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}

	resp, err := http.Get(server.URL + "/api/v1/tasks/" + created.TaskID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got taskResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.TaskID != created.TaskID {
		t.Fatalf("task_id = %q, want %q", got.TaskID, created.TaskID)
	}
}

func TestUnknownTaskReturnsJSONError(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/tasks/missing")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusNotFound)
	}

	var got errorResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.Code != "task_not_found" {
		t.Fatalf("code = %q, want %q", got.Code, "task_not_found")
	}
}
