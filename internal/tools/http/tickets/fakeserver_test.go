package tickets

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFakeHandlerSearchReturnsDeterministicMatch(t *testing.T) {
	server := httptest.NewServer(NewFakeHandler("secret-token"))
	defer server.Close()

	req, err := http.NewRequest(http.MethodGet, server.URL+"/tickets/search?q=database+incident+INC-321", nil)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret-token")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if len(got.Matches) != 1 {
		t.Fatalf("len(Matches) = %d, want %d", len(got.Matches), 1)
	}
	if got.Matches[0].TicketID != "INC-321" {
		t.Fatalf("TicketID = %q, want %q", got.Matches[0].TicketID, "INC-321")
	}
}

func TestFakeHandlerCommentCreateEchoesRequest(t *testing.T) {
	server := httptest.NewServer(NewFakeHandler("secret-token"))
	defer server.Close()

	body := bytes.NewBufferString(`{"comment":"approved note"}`)
	req, err := http.NewRequest(http.MethodPost, server.URL+"/tickets/INC-654/comments", body)
	if err != nil {
		t.Fatalf("NewRequest() error = %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	var got CommentCreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatalf("Decode() error = %v", err)
	}
	if got.TicketID != "INC-654" {
		t.Fatalf("TicketID = %q, want %q", got.TicketID, "INC-654")
	}
	if got.Comment != "approved note" {
		t.Fatalf("Comment = %q, want %q", got.Comment, "approved note")
	}
}

func TestFakeHandlerRejectsMissingAuthorization(t *testing.T) {
	server := httptest.NewServer(NewFakeHandler("secret-token"))
	defer server.Close()

	resp, err := http.Get(server.URL + "/tickets/search?q=INC-100")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("StatusCode = %d, want %d", resp.StatusCode, http.StatusUnauthorized)
	}
}
