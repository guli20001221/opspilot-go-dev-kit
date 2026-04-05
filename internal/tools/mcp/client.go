// Package mcp implements a Model Context Protocol (MCP) client adapter that
// discovers tools from MCP-compliant servers and registers them into the
// OpsPilot tool registry. This enables dynamic tool integration without
// code changes — any MCP server can expose tools that the agent runtime
// can plan against and execute.
//
// The adapter speaks JSON-RPC 2.0 over HTTP, implementing the MCP tools/list
// and tools/call methods. Tool schemas are automatically converted to
// registry.Definition with typed ParameterDef entries.
package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	toolregistry "opspilot-go/internal/tools/registry"
)

// Client connects to an MCP-compliant server over HTTP JSON-RPC.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// ClientOptions configures the MCP client.
type ClientOptions struct {
	BaseURL    string
	Timeout    time.Duration
	HTTPClient *http.Client
}

// NewClient constructs an MCP client for the given server.
func NewClient(opts ClientOptions) *Client {
	client := opts.HTTPClient
	if client == nil {
		timeout := opts.Timeout
		if timeout <= 0 {
			timeout = 10 * time.Second
		}
		client = &http.Client{Timeout: timeout}
	}

	return &Client{
		baseURL:    opts.BaseURL,
		httpClient: client,
	}
}

// --- JSON-RPC 2.0 types ---

type jsonRPCRequest struct {
	JSONRPC string `json:"jsonrpc"`
	ID      int    `json:"id"`
	Method  string `json:"method"`
	Params  any    `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *jsonRPCError   `json:"error,omitempty"`
}

type jsonRPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *jsonRPCError) Error() string {
	return fmt.Sprintf("mcp rpc error %d: %s", e.Code, e.Message)
}

// --- MCP protocol types ---

// MCPTool is one tool definition as returned by an MCP tools/list response.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"` // JSON Schema object
}

// MCPToolListResult is the result of a tools/list call.
type MCPToolListResult struct {
	Tools []MCPTool `json:"tools"`
}

// MCPCallParams is the params for a tools/call request.
type MCPCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

// MCPCallResult is the result of a tools/call response.
type MCPCallResult struct {
	Content []MCPContent `json:"content"`
	IsError bool         `json:"isError,omitempty"`
}

// MCPContent is one content block in a tool call response.
type MCPContent struct {
	Type string `json:"type"` // "text", "image", "resource"
	Text string `json:"text,omitempty"`
}

// --- Client methods ---

// ListTools discovers available tools from the MCP server.
func (c *Client) ListTools(ctx context.Context) ([]MCPTool, error) {
	resp, err := c.call(ctx, "tools/list", nil)
	if err != nil {
		return nil, fmt.Errorf("mcp tools/list: %w", err)
	}

	var result MCPToolListResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("mcp unmarshal tools/list: %w", err)
	}

	return result.Tools, nil
}

// CallTool executes a tool on the MCP server.
func (c *Client) CallTool(ctx context.Context, name string, arguments json.RawMessage) (MCPCallResult, error) {
	resp, err := c.call(ctx, "tools/call", MCPCallParams{
		Name:      name,
		Arguments: arguments,
	})
	if err != nil {
		return MCPCallResult{}, fmt.Errorf("mcp tools/call %s: %w", name, err)
	}

	var result MCPCallResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return MCPCallResult{}, fmt.Errorf("mcp unmarshal tools/call %s: %w", name, err)
	}

	if result.IsError {
		text := "unknown error"
		if len(result.Content) > 0 {
			text = result.Content[0].Text
		}
		return result, fmt.Errorf("mcp tool %s returned error: %s", name, text)
	}

	return result, nil
}

// call performs a JSON-RPC 2.0 request to the MCP server.
func (c *Client) call(ctx context.Context, method string, params any) (json.RawMessage, error) {
	body, err := json.Marshal(jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  method,
		Params:  params,
	})
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http status %d: %s", resp.StatusCode, string(respBody))
	}

	var rpcResp jsonRPCResponse
	if err := json.Unmarshal(respBody, &rpcResp); err != nil {
		return nil, fmt.Errorf("unmarshal rpc response: %w", err)
	}

	if rpcResp.Error != nil {
		return nil, rpcResp.Error
	}

	return rpcResp.Result, nil
}

// --- Registry integration ---

// DiscoverAndRegister connects to the MCP server, discovers all available tools,
// and registers them into the provided tool registry. Each MCP tool is wrapped
// as a registry.Definition with an executor that proxies calls to the MCP server.
// Returns the number of tools registered.
func (c *Client) DiscoverAndRegister(ctx context.Context, registry *toolregistry.Registry) (int, error) {
	tools, err := c.ListTools(ctx)
	if err != nil {
		return 0, err
	}

	for _, tool := range tools {
		def := convertMCPTool(c, tool)
		registry.Register(def)
		slog.Info("registered MCP tool",
			slog.String("tool_name", def.Name),
			slog.String("description", def.Description),
			slog.Int("parameters", len(def.Parameters)),
		)
	}

	return len(tools), nil
}

// convertMCPTool converts an MCP tool definition into a registry.Definition.
func convertMCPTool(client *Client, tool MCPTool) toolregistry.Definition {
	params := extractParameters(tool.InputSchema)

	// MCP tools are treated as side-effecting (write) by default with approval
	// required, since we cannot statically determine their safety class.
	// Override via registry.Register after discovery if specific tools are read-only.
	return toolregistry.Definition{
		Name:             "mcp_" + tool.Name,
		Description:      tool.Description,
		ActionClass:      "write",
		ReadOnly:         false,
		RequiresApproval: true,
		Parameters:       params,
		Executor:         buildMCPExecutor(client, tool.Name),
	}
}

// buildMCPExecutor creates an Executor that proxies tool calls to the MCP server.
func buildMCPExecutor(client *Client, toolName string) toolregistry.Executor {
	return func(ctx context.Context, args json.RawMessage) (any, error) {
		result, err := client.CallTool(ctx, toolName, args)
		if err != nil {
			return nil, err
		}

		// Collect text content from all content blocks
		var texts []string
		for _, content := range result.Content {
			if content.Type == "text" && content.Text != "" {
				texts = append(texts, content.Text)
			}
		}

		return map[string]any{
			"tool":    toolName,
			"content": texts,
		}, nil
	}
}

// extractParameters converts a JSON Schema inputSchema into typed ParameterDef entries.
func extractParameters(schema json.RawMessage) []toolregistry.ParameterDef {
	if len(schema) == 0 {
		return nil
	}

	var parsed struct {
		Type       string                     `json:"type"`
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	if err := json.Unmarshal(schema, &parsed); err != nil {
		return nil
	}
	if parsed.Type != "object" || len(parsed.Properties) == 0 {
		return nil
	}

	requiredSet := make(map[string]bool, len(parsed.Required))
	for _, name := range parsed.Required {
		requiredSet[name] = true
	}

	params := make([]toolregistry.ParameterDef, 0, len(parsed.Properties))
	for name, propRaw := range parsed.Properties {
		var prop struct {
			Type        string `json:"type"`
			Description string `json:"description"`
		}
		if err := json.Unmarshal(propRaw, &prop); err != nil {
			continue
		}
		params = append(params, toolregistry.ParameterDef{
			Name:        name,
			Type:        prop.Type,
			Required:    requiredSet[name],
			Description: prop.Description,
		})
	}

	return params
}
