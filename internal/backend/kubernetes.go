// Package backend implements a Kubernetes backend for running coding agent jobs.
package backend

import (
	"fmt"
	"log/slog"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesBackend struct {
	namespace string
	logger    *slog.Logger
	client    client.Client
}

const DefaultImage = "ghcr.io/manno/background-coder:latest"

func New(cfg *rest.Config, namespace string, logger *slog.Logger) (*KubernetesBackend, error) {
	c, err := NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &KubernetesBackend{
		namespace: namespace,
		logger:    logger,
		client:    c,
	}, nil
}

// Helper functions for pointer values
func boolPtr(b bool) *bool {
	return &b
}

func int32Ptr(i int32) *int32 {
	return &i
}
