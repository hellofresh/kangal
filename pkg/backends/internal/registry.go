package internal

import (
	"errors"

	"github.com/kelseyhightower/envconfig"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

var (
	// ErrBackendRegistered returned when try to register a backend twice
	ErrBackendRegistered = errors.New("Backend already registered")
	// ErrNoBackendRegistered returned when no backend found for given type
	ErrNoBackendRegistered = errors.New("No backend registered")
)

// defaultRegistry contains the list of available backends
var defaultRegistry = map[loadTestV1.LoadTestType]Backend{}

// Register should be called to register your backend
func Register(b Backend) {
	if _, exists := defaultRegistry[b.Type()]; exists {
		panic(ErrBackendRegistered)
	}

	defaultRegistry[b.Type()] = b
}

// Registry you can use this to add information to backends and to resolve to then
type Registry struct {
	registry map[loadTestV1.LoadTestType]Backend
}

// New creates a new Backend instance
func New(opts ...Option) *Registry {
	b := &Registry{
		registry: defaultRegistry,
	}

	for _, opt := range opts {
		opt(b)
	}

	for _, reg := range b.registry {
		if item, ok := reg.(BackendGetEnvConfig); ok {
			err := envconfig.Process("", item.GetEnvConfig())
			if err != nil {
				panic(err)
			}
		}
		if item, ok := reg.(BackendSetDefaults); ok {
			item.SetDefaults()
		}
	}

	return b
}

// Resolve return the given backend name from the registry
func (b *Registry) Resolve(loadTestType loadTestV1.LoadTestType) (Backend, error) {
	resolved, exists := b.registry[loadTestType]
	if !exists {
		return nil, ErrNoBackendRegistered
	}
	return resolved, nil
}
