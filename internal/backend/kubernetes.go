package backend

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/manno/background-coding-agent/internal/change"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesBackend struct {
	namespace string
	logger    *slog.Logger
	client    client.Client
}

func NewKubernetesBackend(cfg *rest.Config, namespace string, logger *slog.Logger) (*KubernetesBackend, error) {
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

func (k *KubernetesBackend) Setup(ctx context.Context) error {
	k.logger.Info("setting up kubernetes backend", "namespace", k.namespace)

	// TODO: Create namespace if not exists
	// TODO: Create secrets for git credentials
	// TODO: Create service account

	return nil
}

func (k *KubernetesBackend) ApplyChange(ctx context.Context, c *change.Change) error {
	k.logger.Info("applying change", "repos", len(c.Spec.Repos))

	for _, repo := range c.Spec.Repos {
		k.logger.Info("creating job for repository", "repo", repo)

		// TODO: Create Kubernetes job for repository
		// TODO: Monitor job status
	}

	return nil
}

func (k *KubernetesBackend) GetJobStatus(ctx context.Context, jobName string) (string, error) {
	// TODO: Query Kubernetes API for job status
	return "unknown", fmt.Errorf("not implemented")
}
