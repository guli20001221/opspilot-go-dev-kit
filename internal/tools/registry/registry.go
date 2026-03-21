package registry

import (
	"sort"
	"sync"
)

// Definition describes one registered tool and its static policy.
type Definition struct {
	Name             string
	ActionClass      string
	ReadOnly         bool
	RequiresApproval bool
	AsyncOnly        bool
	StubResponse     map[string]any
}

// Registry stores tool definitions keyed by name.
type Registry struct {
	mu          sync.RWMutex
	definitions map[string]Definition
}

// New constructs an empty tool registry.
func New() *Registry {
	return &Registry{definitions: make(map[string]Definition)}
}

// Register adds or replaces a tool definition.
func (r *Registry) Register(def Definition) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.definitions[def.Name] = def
}

// Lookup resolves one tool definition by name.
func (r *Registry) Lookup(name string) (Definition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	def, ok := r.definitions[name]
	return def, ok
}

// List returns all registered tool definitions.
func (r *Registry) List() []Definition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]Definition, 0, len(r.definitions))
	for _, def := range r.definitions {
		out = append(out, def)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Name < out[j].Name
	})

	return out
}
