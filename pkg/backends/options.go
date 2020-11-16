package backends

import (
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
	"go.uber.org/zap"

	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"
)

// Option allows to configure backends
type Option func(*Registry)

// WithLogger adds given logger to each registered backend that implements BackendLogger
func WithLogger(logger *zap.Logger) Option {
	return func(b *Registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetLogger); ok {
				iface.SetLogger(logger)
			}
		}
	}
}

// WithPodAnnotations adds given logger to each registered backend that implements BackendLogger
func WithPodAnnotations(podAnnotations map[string]string) Option {
	return func(b *Registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetPodAnnotations); ok {
				iface.SetPodAnnotations(podAnnotations)
			}
		}
	}
}

// WithKubeClientSet adds given kubeClientSet to each registered backend that implements BackendKubeClientSet
func WithKubeClientSet(kubeClientSet kubernetes.Interface) Option {
	return func(b *Registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetKubeClientSet); ok {
				iface.SetKubeClientSet(kubeClientSet)
			}
		}
	}
}

// WithKangalClientSet adds given kangalClientSet to each registered backend that implements BackendKangalClientSet
func WithKangalClientSet(kangalClientSet clientSetV.Interface) Option {
	return func(b *Registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetKangalClientSet); ok {
				iface.SetKangalClientSet(kangalClientSet)
			}
		}
	}
}

// WithNamespaceLister adds given namespaceLister to each registered backend that implements BackendNamespaceLister
func WithNamespaceLister(namespaceLister coreListersV1.NamespaceLister) Option {
	return func(b *Registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetNamespaceLister); ok {
				iface.SetNamespaceLister(namespaceLister)
			}
		}
	}
}
