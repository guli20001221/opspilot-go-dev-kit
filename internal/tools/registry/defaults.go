package registry

import "net/http"

import tickettools "opspilot-go/internal/tools/http/tickets"

// Options configure how the shared default registry binds concrete adapters.
type Options struct {
	TicketAPIBaseURL string
	TicketAPIToken   string
	HTTPClient       *http.Client
}

// NewDefaultRegistry constructs the shared default tool registry used by the
// current chat and workflow skeletons.
func NewDefaultRegistry() *Registry {
	return NewDefaultRegistryWithOptions(Options{})
}

// NewDefaultRegistryWithOptions constructs the shared default registry with
// optional external adapter configuration.
func NewDefaultRegistryWithOptions(opts Options) *Registry {
	registry := New()
	searchExecutor := tickettools.SearchExecutor
	commentExecutor := tickettools.CommentCreateExecutor
	if opts.TicketAPIBaseURL != "" {
		client := tickettools.NewHTTPClient(opts.TicketAPIBaseURL, opts.TicketAPIToken, opts.HTTPClient)
		searchExecutor = client.SearchExecutor
		commentExecutor = client.CommentCreateExecutor
	}

	registry.Register(Definition{
		Name:             "ticket_search",
		Description:      "Search tickets by keyword query",
		ActionClass:      "read",
		ReadOnly:         true,
		RequiresApproval: false,
		Parameters: []ParameterDef{
			{Name: "query", Type: "string", Required: true, Description: "search keywords or ticket ID pattern"},
		},
		Executor: searchExecutor,
	})
	registry.Register(Definition{
		Name:             "ticket_comment_create",
		Description:      "Create a comment on an existing ticket",
		ActionClass:      "write",
		ReadOnly:         false,
		RequiresApproval: true,
		Parameters: []ParameterDef{
			{Name: "ticket_id", Type: "string", Required: true, Description: "ticket identifier (e.g. INC-100)"},
			{Name: "comment", Type: "string", Required: true, Description: "comment text to add"},
		},
		Executor: commentExecutor,
	})

	return registry
}
