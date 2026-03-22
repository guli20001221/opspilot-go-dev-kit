package tickets

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPExecutorsUseConfiguredTicketAPI(t *testing.T) {
	var gotAuth string
	var gotSearchQuery string
	var gotCommentPath string
	var gotCommentBody map[string]string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/tickets/search":
			gotSearchQuery = r.URL.Query().Get("q")
			_ = json.NewEncoder(w).Encode(SearchResponse{
				Matches: []SearchMatch{{TicketID: "INC-900", Summary: "remote result"}},
			})
		case r.Method == http.MethodPost && r.URL.Path == "/tickets/INC-901/comments":
			gotCommentPath = r.URL.Path
			if err := json.NewDecoder(r.Body).Decode(&gotCommentBody); err != nil {
				t.Fatalf("Decode() error = %v", err)
			}
			_ = json.NewEncoder(w).Encode(CommentCreateResponse{
				TicketID: "INC-901",
				Status:   "comment_created",
				Comment:  gotCommentBody["comment"],
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "secret-token", server.Client())

	searchResult, err := client.SearchExecutor(context.Background(), json.RawMessage(`{"query":"INC-900 database issue"}`))
	if err != nil {
		t.Fatalf("SearchExecutor() error = %v", err)
	}
	commentResult, err := client.CommentCreateExecutor(context.Background(), json.RawMessage(`{"ticket_id":"INC-901","comment":"remote comment"}`))
	if err != nil {
		t.Fatalf("CommentCreateExecutor() error = %v", err)
	}

	if gotAuth != "Bearer secret-token" {
		t.Fatalf("Authorization = %q, want %q", gotAuth, "Bearer secret-token")
	}
	if gotSearchQuery != "INC-900 database issue" {
		t.Fatalf("search query = %q, want %q", gotSearchQuery, "INC-900 database issue")
	}
	if gotCommentPath != "/tickets/INC-901/comments" {
		t.Fatalf("comment path = %q, want %q", gotCommentPath, "/tickets/INC-901/comments")
	}
	if gotCommentBody["comment"] != "remote comment" {
		t.Fatalf("comment body = %#v, want comment", gotCommentBody)
	}

	searchPayload, ok := searchResult.(SearchResponse)
	if !ok {
		t.Fatalf("search result type = %T, want %T", searchResult, SearchResponse{})
	}
	if len(searchPayload.Matches) != 1 || searchPayload.Matches[0].TicketID != "INC-900" {
		t.Fatalf("search payload = %#v, want one INC-900 match", searchPayload)
	}

	commentPayload, ok := commentResult.(CommentCreateResponse)
	if !ok {
		t.Fatalf("comment result type = %T, want %T", commentResult, CommentCreateResponse{})
	}
	if commentPayload.TicketID != "INC-901" || commentPayload.Comment != "remote comment" {
		t.Fatalf("comment payload = %#v, want remote INC-901 comment", commentPayload)
	}
}
