package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	toolregistry "opspilot-go/internal/tools/registry"
)

// fakeMCPServer simulates an MCP-compliant JSON-RPC server for testing.
func fakeMCPServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req jsonRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "bad request", http.StatusBadRequest)
			return
		}

		var result any
		switch req.Method {
		case "tools/list":
			result = MCPToolListResult{
				Tools: []MCPTool{
					{
						Name:        "get_weather",
						Description: "Get current weather for a location",
						InputSchema: json.RawMessage(`{"type":"object","properties":{"location":{"type":"string","description":"City name"},"units":{"type":"string","description":"celsius or fahrenheit"}},"required":["location"]}`),
					},
					{
						Name:        "create_issue",
						Description: "Create a GitHub issue",
						InputSchema: json.RawMessage(`{"type":"object","properties":{"title":{"type":"string","description":"Issue title"},"body":{"type":"string","description":"Issue body"},"repo":{"type":"string","description":"Repository name"}},"required":["title","repo"]}`),
					},
				},
			}
		case "tools/call":
			var params MCPCallParams
			raw, _ := json.Marshal(req.Params)
			_ = json.Unmarshal(raw, &params)

			switch params.Name {
			case "get_weather":
				result = MCPCallResult{
					Content: []MCPContent{
						{Type: "text", Text: "72°F, sunny in San Francisco"},
					},
				}
			case "create_issue":
				result = MCPCallResult{
					Content: []MCPContent{
						{Type: "text", Text: "Created issue #42"},
					},
				}
			case "failing_tool":
				result = MCPCallResult{
					IsError: true,
					Content: []MCPContent{
						{Type: "text", Text: "permission denied"},
					},
				}
			default:
				result = MCPCallResult{
					IsError: true,
					Content: []MCPContent{
						{Type: "text", Text: "unknown tool"},
					},
				}
			}
		default:
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(jsonRPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Error:   &jsonRPCError{Code: -32601, Message: "method not found"},
			})
			return
		}

		resultJSON, _ := json.Marshal(result)
		json.NewEncoder(w).Encode(jsonRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Result:  resultJSON,
		})
	}))
}

func TestListTools(t *testing.T) {
	server := fakeMCPServer()
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})
	tools, err := client.ListTools(context.Background())
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools) != 2 {
		t.Fatalf("len(tools) = %d, want 2", len(tools))
	}
	if tools[0].Name != "get_weather" {
		t.Fatalf("tools[0].Name = %q, want %q", tools[0].Name, "get_weather")
	}
	if tools[1].Name != "create_issue" {
		t.Fatalf("tools[1].Name = %q, want %q", tools[1].Name, "create_issue")
	}
}

func TestCallTool(t *testing.T) {
	server := fakeMCPServer()
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})
	result, err := client.CallTool(context.Background(), "get_weather", json.RawMessage(`{"location":"San Francisco"}`))
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if len(result.Content) != 1 || result.Content[0].Text == "" {
		t.Fatalf("result.Content = %v, want non-empty text", result.Content)
	}
}

func TestCallToolReturnsErrorForFailingTool(t *testing.T) {
	server := fakeMCPServer()
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})
	_, err := client.CallTool(context.Background(), "failing_tool", json.RawMessage(`{}`))
	if err == nil {
		t.Fatal("CallTool() error = nil, want error for failing tool")
	}
}

func TestDiscoverAndRegister(t *testing.T) {
	server := fakeMCPServer()
	defer server.Close()

	client := NewClient(ClientOptions{BaseURL: server.URL})
	registry := toolregistry.New()

	count, err := client.DiscoverAndRegister(context.Background(), registry)
	if err != nil {
		t.Fatalf("DiscoverAndRegister() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("registered count = %d, want 2", count)
	}

	// Verify tools are registered with mcp_ prefix
	weather, ok := registry.Lookup("mcp_get_weather")
	if !ok {
		t.Fatal("mcp_get_weather not found in registry")
	}
	if weather.Description != "Get current weather for a location" {
		t.Fatalf("Description = %q, want weather description", weather.Description)
	}
	if weather.ReadOnly {
		t.Fatal("ReadOnly = true, want false (MCP tools default to write)")
	}
	if !weather.RequiresApproval {
		t.Fatal("RequiresApproval = false, want true (MCP tools default to approval)")
	}

	// Verify parameters were extracted from JSON Schema
	if len(weather.Parameters) != 2 {
		t.Fatalf("len(Parameters) = %d, want 2", len(weather.Parameters))
	}
	foundRequired := false
	for _, p := range weather.Parameters {
		if p.Name == "location" && p.Required {
			foundRequired = true
		}
	}
	if !foundRequired {
		t.Fatal("location parameter not found or not required")
	}

	// Verify executor works — proxies to MCP server
	result, err := weather.Executor(context.Background(), json.RawMessage(`{"location":"SF"}`))
	if err != nil {
		t.Fatalf("Executor() error = %v", err)
	}
	data, _ := json.Marshal(result)
	if len(data) == 0 {
		t.Fatal("Executor() returned empty result")
	}
}

func TestExtractParameters(t *testing.T) {
	schema := json.RawMessage(`{
		"type": "object",
		"properties": {
			"query": {"type": "string", "description": "Search query"},
			"limit": {"type": "number", "description": "Max results"}
		},
		"required": ["query"]
	}`)

	params := extractParameters(schema)
	if len(params) != 2 {
		t.Fatalf("len(params) = %d, want 2", len(params))
	}

	paramMap := make(map[string]toolregistry.ParameterDef)
	for _, p := range params {
		paramMap[p.Name] = p
	}

	if q, ok := paramMap["query"]; !ok || !q.Required || q.Type != "string" {
		t.Fatalf("query param = %+v, want required string", paramMap["query"])
	}
	if l, ok := paramMap["limit"]; !ok || l.Required || l.Type != "number" {
		t.Fatalf("limit param = %+v, want optional number", paramMap["limit"])
	}
}

func TestExtractParametersHandlesEmptySchema(t *testing.T) {
	params := extractParameters(nil)
	if params != nil {
		t.Fatalf("extractParameters(nil) = %v, want nil", params)
	}

	params = extractParameters(json.RawMessage(`{}`))
	if params != nil {
		t.Fatalf("extractParameters({}) = %v, want nil", params)
	}
}
