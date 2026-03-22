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
		ActionClass:      "read",
		ReadOnly:         true,
		RequiresApproval: false,
		Executor:         searchExecutor,
	})
	registry.Register(Definition{
		Name:             "ticket_comment_create",
		ActionClass:      "write",
		ReadOnly:         false,
		RequiresApproval: true,
		Executor:         commentExecutor,
	})

	return registry
}
