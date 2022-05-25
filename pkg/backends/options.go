package backends

import (
	"go.uber.org/zap"
	kubeCoreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"

	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

// Option allows to configure backends
type Option func(*registry)

// WithLogger adds given logger to each registered backend that implements BackendSetLogger
func WithLogger(logger *zap.Logger) Option {
	return func(b *registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetLogger); ok {
				iface.SetLogger(logger)
			}
		}
	}
}

// WithPodAnnotations adds given pod annotations to each registered backend that implements BackendSetPodAnnotations
func WithPodAnnotations(podAnnotations map[string]string) Option {
	return func(b *registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetPodAnnotations); ok {
				iface.SetPodAnnotations(podAnnotations)
			}
		}
	}
}

// WithNodeSelector adds given pod node selectors to each registered backend that implements BackendSetPodNodeSelector
func WithNodeSelector(nodeSelector map[string]string) Option {
	return func(b *registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetPodNodeSelector); ok {
				iface.SetPodNodeSelector(nodeSelector)
			}
		}
	}
}

// WithTolerations adds given pod tolerations to each registered backend that implements BackendSetPodTolerations
func WithTolerations(tolerations []kubeCoreV1.Toleration) Option {
	return func(b *registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetPodTolerations); ok {
				iface.SetPodTolerations(tolerations)
			}
		}
	}
}

// WithKubeClientSet adds given kubeClientSet to each registered backend that implements BackendKubeClientSet
func WithKubeClientSet(kubeClientSet kubernetes.Interface) Option {
	return func(b *registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetKubeClientSet); ok {
				iface.SetKubeClientSet(kubeClientSet)
			}
		}
	}
}

// WithKangalClientSet adds given kangalClientSet to each registered backend that implements BackendKangalClientSet
func WithKangalClientSet(kangalClientSet clientSetV.Interface) Option {
	return func(b *registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetKangalClientSet); ok {
				iface.SetKangalClientSet(kangalClientSet)
			}
		}
	}
}

// WithNamespaceLister adds given namespaceLister to each registered backend that implements BackendNamespaceLister
func WithNamespaceLister(namespaceLister coreListersV1.NamespaceLister) Option {
	return func(b *registry) {
		for _, item := range b.registry {
			if iface, ok := item.(BackendSetNamespaceLister); ok {
				iface.SetNamespaceLister(namespaceLister)
			}
		}
	}
}
