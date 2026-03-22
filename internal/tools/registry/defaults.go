package registry

import tickettools "opspilot-go/internal/tools/http/tickets"

// NewDefaultRegistry constructs the shared default tool registry used by the
// current chat and workflow skeletons.
func NewDefaultRegistry() *Registry {
	registry := New()
	registry.Register(Definition{
		Name:             "ticket_search",
		ActionClass:      "read",
		ReadOnly:         true,
		RequiresApproval: false,
		Executor:         tickettools.SearchExecutor,
	})
	registry.Register(Definition{
		Name:             "ticket_comment_create",
		ActionClass:      "write",
		ReadOnly:         false,
		RequiresApproval: true,
		Executor:         tickettools.CommentCreateExecutor,
	})

	return registry
}
