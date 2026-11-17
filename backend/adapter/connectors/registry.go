package connectors

import (
	"fmt"
	"sort"
	"sync"
)

// Registry holds the connectors one B process offers. The composition root
// adds them; the core looks them up by the id stored on a connection.
type Registry struct {
	mu         sync.RWMutex
	connectors map[string]Connector
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{connectors: map[string]Connector{}}
}

// Register adds c after checking its manifest and credential spec. It refuses
// a second connector with the same id.
func (r *Registry) Register(c Connector) error {
	id := c.ID()
	m := c.Manifest()
	if m.Connector != id {
		return fmt.Errorf("connector %q: manifest names %q", id, m.Connector)
	}
	if err := m.Check(); err != nil {
		return fmt.Errorf("connector %q: manifest: %w", id, err)
	}
	if err := c.Credentials().Check(); err != nil {
		return fmt.Errorf("connector %q: credentials: %w", id, err)
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, dup := r.connectors[id]; dup {
		return fmt.Errorf("connector %q is registered twice", id)
	}
	r.connectors[id] = c
	return nil
}

// Get returns the connector with this id.
func (r *Registry) Get(id string) (Connector, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	c, ok := r.connectors[id]
	return c, ok
}

// All returns every connector, sorted by id.
func (r *Registry) All() []Connector {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]Connector, 0, len(r.connectors))
	for _, c := range r.connectors {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID() < out[j].ID() })
	return out
}
