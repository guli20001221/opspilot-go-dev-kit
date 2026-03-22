package tickets

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"
)

var fakeCommentPathPattern = regexp.MustCompile(`^/tickets/([^/]+)/comments$`)

// NewFakeHandler constructs a deterministic in-repo ticket API for local development.
func NewFakeHandler(expectedToken string) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/tickets/search", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeRequest(w, r, expectedToken) {
			return
		}
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
		if query == "" {
			http.Error(w, "missing query", http.StatusBadRequest)
			return
		}

		ticketID := strings.ToUpper(ticketIDPattern.FindString(query))
		if ticketID == "" {
			ticketID = "INC-100"
		}

		writeJSON(w, SearchResponse{
			Matches: []SearchMatch{
				{
					TicketID: ticketID,
					Summary:  "fake ticket match for " + query,
				},
			},
		})
	})
	mux.HandleFunc("/tickets/", func(w http.ResponseWriter, r *http.Request) {
		if !authorizeRequest(w, r, expectedToken) {
			return
		}
		if r.Method != http.MethodPost {
			http.NotFound(w, r)
			return
		}

		matches := fakeCommentPathPattern.FindStringSubmatch(r.URL.Path)
		if len(matches) != 2 {
			http.NotFound(w, r)
			return
		}

		var body struct {
			Comment string `json:"comment"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, "invalid json", http.StatusBadRequest)
			return
		}
		body.Comment = strings.TrimSpace(body.Comment)
		if body.Comment == "" {
			http.Error(w, "missing comment", http.StatusBadRequest)
			return
		}

		writeJSON(w, CommentCreateResponse{
			TicketID: strings.ToUpper(matches[1]),
			Status:   "comment_created",
			Comment:  body.Comment,
		})
	})

	return mux
}

func authorizeRequest(w http.ResponseWriter, r *http.Request, expectedToken string) bool {
	if expectedToken == "" {
		return true
	}
	if r.Header.Get("Authorization") != "Bearer "+expectedToken {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return false
	}

	return true
}

func writeJSON(w http.ResponseWriter, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(payload)
}
