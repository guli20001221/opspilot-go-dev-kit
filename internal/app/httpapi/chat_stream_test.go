package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestChatStreamReturnsSSEAndPersistsMessages(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	body := bytes.NewBufferString(`{"tenant_id":"tenant-1","user_id":"user-1","mode":"chat","user_message":"hello"}`)
	resp, err := http.Post(server.URL+"/api/v1/chat/stream", "application/json", body)
	if err != nil {
		t.Fatalf("Post() error = %v", err)
	}
	defer resp.Body.Close()

	if got := resp.Header.Get("Content-Type"); !strings.HasPrefix(got, "text/event-stream") {
		t.Fatalf("Content-Type = %q, want prefix %q", got, "text/event-stream")
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("ReadAll() error = %v", err)
	}
	text := string(raw)

	metaIdx := strings.Index(text, "event: meta")
	planIdx := strings.Index(text, "event: plan")
	retrievalIdx := strings.Index(text, "event: retrieval")
	stateIdx := strings.Index(text, "event: state")
	doneIdx := strings.Index(text, "event: done")
	if metaIdx == -1 || planIdx == -1 || retrievalIdx == -1 || stateIdx == -1 || doneIdx == -1 {
		t.Fatalf("missing expected events in stream: %q", text)
	}
	if !(metaIdx < planIdx && planIdx < retrievalIdx && retrievalIdx < stateIdx && stateIdx < doneIdx) {
		t.Fatalf("unexpected event order: %q", text)
	}

	metaPayload := extractEventPayload(t, text, "meta")
	if resp.Header.Get(requestIDHeader) == "" {
		t.Fatal("X-Request-Id header is empty")
	}
	if resp.Header.Get(traceIDHeader) == "" {
		t.Fatal("X-Trace-Id header is empty")
	}
	if metaPayload["request_id"] != resp.Header.Get(requestIDHeader) {
		t.Fatalf("meta request_id = %q, want %q", metaPayload["request_id"], resp.Header.Get(requestIDHeader))
	}
	if metaPayload["trace_id"] != resp.Header.Get(traceIDHeader) {
		t.Fatalf("meta trace_id = %q, want %q", metaPayload["trace_id"], resp.Header.Get(traceIDHeader))
	}
	if metaPayload["request_id"] == "" {
		t.Fatal("meta request_id is empty")
	}
	if metaPayload["trace_id"] == "" {
		t.Fatal("meta trace_id is empty")
	}

	planPayload := extractEventPayload(t, text, "plan")
	if planPayload["intent"] != "knowledge_qa" {
		t.Fatalf("plan intent = %q, want %q", planPayload["intent"], "knowledge_qa")
	}

	retrievalPayload := extractEventPayload(t, text, "retrieval")
	if retrievalPayload["query_used"] != "hello" {
		t.Fatalf("retrieval query_used = %q, want %q", retrievalPayload["query_used"], "hello")
	}

	sessionID := extractSessionID(t, text)

	msgResp, err := http.Get(server.URL + "/api/v1/sessions/" + sessionID + "/messages")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer msgResp.Body.Close()

	var got struct {
		Messages []messageResponse `json:"messages"`
	}
	if err := json.NewDecoder(msgResp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Messages) != 2 {
		t.Fatalf("len(Messages) = %d, want %d", len(got.Messages), 2)
	}
	if got.Messages[0].Role != "user" || got.Messages[1].Role != "assistant" {
		t.Fatalf("unexpected roles = %#v", got.Messages)
	}
}

func TestChatStreamEmitsToolAndTaskPromotedEventsWhenApplicable(t *testing.T) {
	server := httptest.NewServer(NewHandler())
	defer server.Close()

	toolResp, err := http.Post(server.URL+"/api/v1/chat/stream", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-1","user_id":"user-1","mode":"chat","user_message":"search related ticket history"}`))
	if err != nil {
		t.Fatalf("tool Post() error = %v", err)
	}
	defer toolResp.Body.Close()

	toolRaw, err := io.ReadAll(toolResp.Body)
	if err != nil {
		t.Fatalf("tool ReadAll() error = %v", err)
	}
	toolPayload := extractEventPayload(t, string(toolRaw), "tool")
	if toolPayload["tool_name"] != "ticket_search" {
		t.Fatalf("tool_name = %q, want %q", toolPayload["tool_name"], "ticket_search")
	}
	if toolPayload["status"] != "succeeded" {
		t.Fatalf("tool status = %q, want %q", toolPayload["status"], "succeeded")
	}

	taskResp, err := http.Post(server.URL+"/api/v1/chat/stream", "application/json", bytes.NewBufferString(`{"tenant_id":"tenant-1","user_id":"user-1","mode":"task","user_message":"generate a report for last week's incidents"}`))
	if err != nil {
		t.Fatalf("task Post() error = %v", err)
	}
	defer taskResp.Body.Close()

	taskRaw, err := io.ReadAll(taskResp.Body)
	if err != nil {
		t.Fatalf("task ReadAll() error = %v", err)
	}
	taskPayload := extractEventPayload(t, string(taskRaw), "task_promoted")
	if taskPayload["status"] != "queued" {
		t.Fatalf("task status = %q, want %q", taskPayload["status"], "queued")
	}
	if taskPayload["reason"] != "workflow_required" {
		t.Fatalf("task reason = %q, want %q", taskPayload["reason"], "workflow_required")
	}
}

func extractEventPayload(t *testing.T, stream string, eventName string) map[string]string {
	t.Helper()

	lines := strings.Split(stream, "\n")
	for i := 0; i < len(lines)-1; i++ {
		if lines[i] != "event: "+eventName {
			continue
		}
		if !strings.HasPrefix(lines[i+1], "data: ") {
			t.Fatalf("event %q missing data line in stream: %q", eventName, stream)
		}

		var payload map[string]string
		if err := json.Unmarshal([]byte(strings.TrimPrefix(lines[i+1], "data: ")), &payload); err != nil {
			t.Fatalf("Unmarshal(%q payload) error = %v", eventName, err)
		}
		return payload
	}

	t.Fatalf("event %q not found in SSE stream", eventName)
	return nil
}

func extractSessionID(t *testing.T, stream string) string {
	t.Helper()

	payload := extractEventPayload(t, stream, "meta")
	if payload["session_id"] != "" {
		return payload["session_id"]
	}

	t.Fatal("session_id not found in SSE stream")
	return ""
}
