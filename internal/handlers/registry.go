package handlers

import (
	"sync"
)

// Registry manages all resource handlers
type Registry struct {
	mu       sync.RWMutex
	handlers map[string]ResourceHandler
	aliases  map[string]string
	order    []string // Maintains registration order for display
}

// NewRegistry creates a new handler registry
func NewRegistry() *Registry {
	return &Registry{
		handlers: make(map[string]ResourceHandler),
		aliases:  make(map[string]string),
		order:    make([]string, 0),
	}
}

// Register adds a handler to the registry
func (r *Registry) Register(handler ResourceHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()

	resourceType := handler.ResourceType()
	r.handlers[resourceType] = handler
	r.order = append(r.order, resourceType)

	// Also register by shortcut key
	if key := handler.ShortcutKey(); key != "" {
		r.aliases[key] = resourceType
	}

	// Register by resource name (lowercase)
	r.aliases[handler.ResourceName()] = resourceType
}

// Get retrieves a handler by type, alias, or shortcut
func (r *Registry) Get(typeOrAlias string) (ResourceHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Try direct lookup
	if h, ok := r.handlers[typeOrAlias]; ok {
		return h, true
	}

	// Try alias lookup
	if actual, ok := r.aliases[typeOrAlias]; ok {
		return r.handlers[actual], true
	}

	return nil, false
}

// All returns all registered handlers in registration order
func (r *Registry) All() []ResourceHandler {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]ResourceHandler, 0, len(r.order))
	for _, resourceType := range r.order {
		if h, ok := r.handlers[resourceType]; ok {
			result = append(result, h)
		}
	}
	return result
}

// Types returns all registered resource types
func (r *Registry) Types() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]string, len(r.order))
	copy(result, r.order)
	return result
}

// GetByShortcut retrieves a handler by its shortcut key
func (r *Registry) GetByShortcut(shortcut string) (ResourceHandler, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if actual, ok := r.aliases[shortcut]; ok {
		return r.handlers[actual], true
	}
	return nil, false
}

// Shortcuts returns a map of shortcut keys to resource types
func (r *Registry) Shortcuts() map[string]string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]string)
	for alias, resourceType := range r.aliases {
		result[alias] = resourceType
	}
	return result
}
