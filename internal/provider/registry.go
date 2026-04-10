package provider

import "fmt"

type Registry struct {
	providers map[string]Provider
}

// NewRegistry creates a new provider registry
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]Provider),
	}
}

// Register registers a provider
func (r *Registry) Register(provider Provider) {
	r.providers[provider.ID()] = provider
}

// Get retrieves a provider by ID
func (r *Registry) Get(id string) (Provider, error) {
	p, ok := r.providers[id]
	if !ok {
		return nil, fmt.Errorf("provider not found: %s", id)
	}
	return p, nil
}

// List returns all registered providers
func (r *Registry) List() []Provider {
	var providers []Provider
	for _, p := range r.providers {
		providers = append(providers, p)
	}
	return providers
}

// Exists checks if a provider is registered
func (r *Registry) Exists(id string) bool {
	_, ok := r.providers[id]
	return ok
}
