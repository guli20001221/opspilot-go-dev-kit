package tickets

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// HTTPClient executes ticket adapters over an HTTP boundary.
type HTTPClient struct {
	baseURL string
	token   string
	client  *http.Client
}

// NewHTTPClient constructs a ticket API adapter backed by the provided base URL.
func NewHTTPClient(baseURL string, token string, client *http.Client) *HTTPClient {
	if client == nil {
		client = http.DefaultClient
	}

	return &HTTPClient{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		client:  client,
	}
}

// SearchExecutor calls the configured ticket search endpoint.
func (c *HTTPClient) SearchExecutor(ctx context.Context, args json.RawMessage) (any, error) {
	var payload SearchRequest
	if err := json.Unmarshal(args, &payload); err != nil {
		return nil, fmt.Errorf("decode ticket_search arguments: %w", err)
	}
	payload.Query = strings.TrimSpace(payload.Query)
	if payload.Query == "" {
		return nil, fmt.Errorf("ticket_search requires query")
	}

	endpoint := c.baseURL + "/tickets/search?q=" + url.QueryEscape(payload.Query)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("build ticket_search request: %w", err)
	}
	c.setHeaders(req)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ticket_search: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("call ticket_search: status %d", resp.StatusCode)
	}

	var out SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode ticket_search response: %w", err)
	}

	return out, nil
}

// CommentCreateExecutor calls the configured ticket comment endpoint.
func (c *HTTPClient) CommentCreateExecutor(ctx context.Context, args json.RawMessage) (any, error) {
	var payload CommentCreateRequest
	if err := json.Unmarshal(args, &payload); err != nil {
		return nil, fmt.Errorf("decode ticket_comment_create arguments: %w", err)
	}
	payload.TicketID = strings.ToUpper(strings.TrimSpace(payload.TicketID))
	payload.Comment = strings.TrimSpace(payload.Comment)
	if payload.TicketID == "" {
		return nil, fmt.Errorf("ticket_comment_create requires ticket_id")
	}
	if payload.Comment == "" {
		return nil, fmt.Errorf("ticket_comment_create requires comment")
	}

	body, err := json.Marshal(map[string]string{"comment": payload.Comment})
	if err != nil {
		return nil, fmt.Errorf("marshal ticket_comment_create request: %w", err)
	}

	endpoint := c.baseURL + "/tickets/" + url.PathEscape(payload.TicketID) + "/comments"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("build ticket_comment_create request: %w", err)
	}
	c.setHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call ticket_comment_create: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("call ticket_comment_create: status %d", resp.StatusCode)
	}

	var out CommentCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("decode ticket_comment_create response: %w", err)
	}

	return out, nil
}

func (c *HTTPClient) setHeaders(req *http.Request) {
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
}
