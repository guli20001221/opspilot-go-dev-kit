package tickets

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var ticketIDPattern = regexp.MustCompile(`(?i)\b[A-Z]+-\d+\b`)

// SearchRequest is the typed request for the ticket search adapter.
type SearchRequest struct {
	Query string `json:"query"`
}

// SearchMatch is one deterministic ticket search hit.
type SearchMatch struct {
	TicketID string `json:"ticket_id"`
	Summary  string `json:"summary"`
}

// SearchResponse is the normalized ticket search result.
type SearchResponse struct {
	Matches []SearchMatch `json:"matches"`
}

// CommentCreateRequest is the typed request for ticket comment creation.
type CommentCreateRequest struct {
	TicketID string `json:"ticket_id"`
	Comment  string `json:"comment"`
}

// CommentCreateResponse is the normalized ticket comment result.
type CommentCreateResponse struct {
	TicketID string `json:"ticket_id"`
	Status   string `json:"status"`
	Comment  string `json:"comment"`
}

// SearchExecutor runs the deterministic ticket search adapter.
func SearchExecutor(_ context.Context, args json.RawMessage) (any, error) {
	var payload SearchRequest
	if err := json.Unmarshal(args, &payload); err != nil {
		return nil, fmt.Errorf("decode ticket_search arguments: %w", err)
	}
	payload.Query = strings.TrimSpace(payload.Query)
	if payload.Query == "" {
		return nil, fmt.Errorf("ticket_search requires query")
	}

	ticketID := strings.ToUpper(ticketIDPattern.FindString(payload.Query))
	if ticketID == "" {
		ticketID = "INC-100"
	}

	return SearchResponse{
		Matches: []SearchMatch{
			{
				TicketID: ticketID,
				Summary:  fmt.Sprintf("deterministic match for %s", payload.Query),
			},
		},
	}, nil
}

// CommentCreateExecutor runs the deterministic ticket comment adapter.
func CommentCreateExecutor(_ context.Context, args json.RawMessage) (any, error) {
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

	return CommentCreateResponse{
		TicketID: payload.TicketID,
		Status:   "comment_created",
		Comment:  payload.Comment,
	}, nil
}
