package backend

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (k *KubernetesBackend) Setup(ctx context.Context, credentials map[string]string) error {
	k.logger.Info("setting up kubernetes backend", "namespace", k.namespace)

	// Create namespace if it doesn't exist
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: k.namespace,
		},
	}

	if err := k.client.Create(ctx, ns); err != nil {
		// Check if it already exists
		if err := k.client.Get(ctx, client.ObjectKey{Name: k.namespace}, ns); err != nil {
			return fmt.Errorf("failed to create or get namespace: %w", err)
		}
		k.logger.Info("namespace already exists", "namespace", k.namespace)
	} else {
		k.logger.Info("namespace created", "namespace", k.namespace)
	}

	// Create secret with all provided credentials
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bca-credentials",
			Namespace: k.namespace,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: credentials,
	}

	k.logger.Info("storing credentials", "count", len(credentials))

	if err := k.client.Create(ctx, secret); err != nil {
		// Check if it already exists and update
		existingSecret := &corev1.Secret{}
		if err := k.client.Get(ctx, client.ObjectKey{Name: secret.Name, Namespace: k.namespace}, existingSecret); err != nil {
			return fmt.Errorf("failed to create or get secret: %w", err)
		}

		// Update existing secret
		existingSecret.StringData = secret.StringData
		if err := k.client.Update(ctx, existingSecret); err != nil {
			return fmt.Errorf("failed to update secret: %w", err)
		}
		k.logger.Info("secret updated", "name", secret.Name)
	} else {
		k.logger.Info("secret created", "name", secret.Name)
	}

	return nil
}
