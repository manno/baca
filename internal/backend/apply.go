package backend

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/manno/background-coding-agent/internal/agent"
	"github.com/manno/background-coding-agent/internal/change"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

	container := corev1.Container{
		Name:            "runner",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
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
	}

	podSpec := corev1.PodSpec{
		RestartPolicy: corev1.RestartPolicyNever,
		Containers:    []corev1.Container{container},
	}

	// Mount gemini OAuth files if using gemini-cli
	if c.Spec.Agent == "gemini-cli" {
		// Check if we have gemini OAuth files in the secret (not API key)
		container.VolumeMounts = []corev1.VolumeMount{
			{
				Name:      "gemini-oauth",
				MountPath: "/root/.gemini",
				ReadOnly:  true,
			},
		}
		podSpec.Containers[0] = container

		podSpec.Volumes = []corev1.Volume{
			{
				Name: "gemini-oauth",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: "bca-credentials",
						Items: []corev1.KeyToPath{
							{Key: "GEMINI_oauth_creds.json", Path: "oauth_creds.json", Mode: int32Ptr(0600)},
							{Key: "GEMINI_google_accounts.json", Path: "google_accounts.json", Mode: int32Ptr(0600)},
							{Key: "GEMINI_installation_id", Path: "installation_id", Mode: int32Ptr(0600)},
							{Key: "GEMINI_settings.json", Path: "settings.json", Mode: int32Ptr(0600)},
						},
						Optional: boolPtr(true), // Optional in case using API key instead
					},
				},
			},
		}
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":                          "background-coding-agent",
				"app.kubernetes.io/name":       "bca",
				"app.kubernetes.io/component":  "job",
				"app.kubernetes.io/managed-by": "bca-cli",
				"repo":                         k.sanitizeLabel(repoURL),
			},
		},
		Spec: batchv1.JobSpec{
			// Automatically clean up jobs after completion
			TTLSecondsAfterFinished: int32Ptr(3600), // Clean up after 1 hour
			BackoffLimit:            int32Ptr(3),    // Retry up to 3 times on failure
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}

	return job
}

func (k *KubernetesBackend) buildJobScript(c *change.Change, repoURL string) string {
	// Get agent command from configuration
	agentCommand := agent.GetCommand(c.Spec.Agent)

	script := []string{
		"set -e",
		"cd /workspace",
		// Configure git to use GITHUB_TOKEN for authentication
		"git config --global credential.helper store",
		"echo \"https://x-access-token:${GITHUB_TOKEN}@github.com\" > ~/.git-credentials",
		"chmod 600 ~/.git-credentials",
		"git config --global user.email \"bca@example.com\"",
		"git config --global user.name \"BCA Bot\"",
		"fleet gitcloner $REPO_URL ./repo",
		"cd ./repo",
	}

	// Download agents.md and resources
	if c.Spec.AgentsMD != "" {
		script = append(script, fmt.Sprintf("curl -L -o agents.md '%s'", c.Spec.AgentsMD))
	}

	for i, res := range c.Spec.Resources {
		script = append(script, fmt.Sprintf("curl -L -o resource-%d.md '%s'", i, res))
	}

	// Execute the coding agent with the mapped command
	// For copilot-cli, prefer COPILOT_TOKEN if available, otherwise use GITHUB_TOKEN
	if c.Spec.Agent == "copilot-cli" {
		script = append(script, "export GITHUB_TOKEN=${COPILOT_TOKEN:-$GITHUB_TOKEN}")
	}
	script = append(script, fmt.Sprintf("%s \"$PROMPT\"", agentCommand))

	// Create pull request (restore GITHUB_TOKEN for gh CLI if we changed it)
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
