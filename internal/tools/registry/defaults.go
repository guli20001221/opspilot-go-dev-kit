package registry

// NewDefaultRegistry constructs the shared default tool registry used by the
// current chat and workflow skeletons.
func NewDefaultRegistry() *Registry {
	registry := New()
	registry.Register(Definition{
		Name:             "ticket_search",
		ActionClass:      "read",
		ReadOnly:         true,
		RequiresApproval: false,
		StubResponse: map[string]any{
			"matches": []map[string]string{
				{"ticket_id": "INC-100", "summary": "database incident"},
			},
		},
	})
	registry.Register(Definition{
		Name:             "ticket_comment_create",
		ActionClass:      "write",
		ReadOnly:         false,
		RequiresApproval: true,
		StubResponse: map[string]any{
			"ticket_id": "INC-100",
			"status":    "comment_created",
		},
	})

	return registry
}
