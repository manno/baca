// Package backend implements a Kubernetes backend for running coding agent jobs.
package backend

import (
	"context"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"time"

	"github.com/manno/background-coding-agent/internal/change"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type KubernetesBackend struct {
	namespace string
	logger    *slog.Logger
	client    client.Client
}

const DefaultImage = "ghcr.io/manno/background-coding-agent:latest"

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

func (k *KubernetesBackend) Setup(ctx context.Context, githubToken string) error {
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

	// Create secret with credentials
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bca-credentials",
			Namespace: k.namespace,
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			"GITHUB_TOKEN": githubToken,
		},
	}

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

func (k *KubernetesBackend) ApplyChange(ctx context.Context, c *change.Change, wait bool) error {
	k.logger.Info("applying change", "repos", len(c.Spec.Repos))

	var jobNames []string
	for _, repo := range c.Spec.Repos {
		k.logger.Info("creating job for repository", "repo", repo)

		job := k.createJob(c, repo)

		if err := k.client.Create(ctx, job); err != nil {
			k.logger.Error("failed to create job in kubernetes", "repo", repo, "error", err)
			return fmt.Errorf("failed to create kubernetes job for %s: %w", repo, err)
		}

		k.logger.Info("job created", "repo", repo, "job", job.Name)
		jobNames = append(jobNames, job.Name)
	}

	// Monitor job status if requested
	if wait {
		k.logger.Info("monitoring jobs", "count", len(jobNames))
		return k.monitorJobs(ctx, jobNames)
	}

	return nil
}

func (k *KubernetesBackend) createJob(c *change.Change, repoURL string) *batchv1.Job {
	jobName := k.generateJobName(repoURL)
	image := c.Spec.Image
	if image == "" {
		image = DefaultImage
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":  "background-coding-agent",
				"repo": k.sanitizeLabel(repoURL),
			},
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					RestartPolicy: corev1.RestartPolicyNever,
					Containers: []corev1.Container{
						{
							Name:  "runner",
							Image: image,
							Command: []string{
								"/bin/sh",
								"-c",
								k.buildJobScript(c, repoURL),
							},
							Env: []corev1.EnvVar{
								{
									Name:  "REPO_URL",
									Value: repoURL,
								},
								{
									Name:  "AGENT",
									Value: c.Spec.Agent,
								},
								{
									Name:  "AGENTS_MD",
									Value: c.Spec.AgentsMD,
								},
								{
									Name:  "PROMPT",
									Value: c.Spec.Prompt,
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									SecretRef: &corev1.SecretEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "bca-credentials",
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	return job
}

func (k *KubernetesBackend) buildJobScript(c *change.Change, repoURL string) string {
	script := []string{
		"set -e",
		"cd /workspace",
		"bca clone $REPO_URL --output ./repo",
		"cd ./repo",
	}

	// Download agents.md and resources
	if c.Spec.AgentsMD != "" {
		script = append(script, fmt.Sprintf("curl -L -o agents.md '%s'", c.Spec.AgentsMD))
	}

	for i, res := range c.Spec.Resources {
		script = append(script, fmt.Sprintf("curl -L -o resource-%d.md '%s'", i, res))
	}

	// Execute the coding agent
	script = append(script, fmt.Sprintf("%s \"$PROMPT\"", c.Spec.Agent))

	// Create pull request
	script = append(script, "gh pr create --fill")

	return strings.Join(script, " && ")
}

func (k *KubernetesBackend) generateJobName(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil {
		return "bca-job"
	}

	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.ToLower(path)

	if len(path) > 50 {
		path = path[:50]
	}

	return fmt.Sprintf("bca-%s", path)
}

func (k *KubernetesBackend) sanitizeLabel(s string) string {
	s = strings.ReplaceAll(s, "https://", "")
	s = strings.ReplaceAll(s, "http://", "")
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, ".", "-")
	s = strings.ToLower(s)

	if len(s) > 63 {
		s = s[:63]
	}

	return s
}

func (k *KubernetesBackend) GetJobStatus(ctx context.Context, jobName string) (string, error) {
	job := &batchv1.Job{}
	if err := k.client.Get(ctx, client.ObjectKey{Name: jobName, Namespace: k.namespace}, job); err != nil {
		return "", fmt.Errorf("failed to get job: %w", err)
	}

	// Check job conditions
	for _, condition := range job.Status.Conditions {
		if condition.Type == batchv1.JobComplete && condition.Status == corev1.ConditionTrue {
			return "Complete", nil
		}
		if condition.Type == batchv1.JobFailed && condition.Status == corev1.ConditionTrue {
			return "Failed", nil
		}
	}

	if job.Status.Active > 0 {
		return "Running", nil
	}

	return "Pending", nil
}

func (k *KubernetesBackend) monitorJobs(ctx context.Context, jobNames []string) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	timeout := time.After(30 * time.Minute)
	jobStatus := make(map[string]string)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timeout waiting for jobs to complete")
		case <-ticker.C:
			allDone := true
			anyFailed := false

			for _, jobName := range jobNames {
				if jobStatus[jobName] == "Complete" || jobStatus[jobName] == "Failed" {
					continue
				}

				status, err := k.GetJobStatus(ctx, jobName)
				if err != nil {
					k.logger.Error("failed to get job status", "job", jobName, "error", err)
					continue
				}

				if status != jobStatus[jobName] {
					k.logger.Info("job status changed", "job", jobName, "status", status)
					jobStatus[jobName] = status
				}

				if status != "Complete" && status != "Failed" {
					allDone = false
				}
				if status == "Failed" {
					anyFailed = true
				}
			}

			if allDone {
				if anyFailed {
					k.logger.Error("some jobs failed")
					k.logJobSummary(jobStatus)
					return fmt.Errorf("some jobs failed")
				}
				k.logger.Info("all jobs completed successfully")
				k.logJobSummary(jobStatus)
				return nil
			}
		}
	}
}

func (k *KubernetesBackend) logJobSummary(jobStatus map[string]string) {
	k.logger.Info("job summary")
	for job, status := range jobStatus {
		k.logger.Info("job status", "job", job, "status", status)
	}
}
