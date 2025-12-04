// Package backend implements a Kubernetes backend for running coding agent jobs.
package k8s

import (
	"fmt"
	"log/slog"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesBackend struct {
	namespace string
	logger    *slog.Logger
	client    client.Client
	clientset *kubernetes.Clientset
}

const DefaultImage = "ghcr.io/manno/background-coder:latest"

func New(cfg *rest.Config, namespace string, logger *slog.Logger) (*KubernetesBackend, error) {
	c, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %w", err)
	}

	return &KubernetesBackend{
		namespace: namespace,
		logger:    logger,
		client:    c,
		clientset: clientset,
	}, nil
}

// Helper functions for pointer values
func boolPtr(b bool) *bool {
	return &b
}

func int32Ptr(i int32) *int32 {
	return &i
}
