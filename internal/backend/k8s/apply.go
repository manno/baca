package k8s

import (
	"bufio"
	"context"
	"crypto/rand"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/manno/baca/internal/change"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

//go:embed scripts/fork-setup.sh
var forkSetupScript string

//go:embed scripts/job-runner.sh
var jobScript string

func (k *KubernetesBackend) ApplyChange(ctx context.Context, c *change.Change, wait bool, retries int32, forkOrg string) error {
	k.logger.Info("applying change", "repos", len(c.Spec.Repos), "fork-org", forkOrg)

	var jobNames []string
	for _, repo := range c.Spec.Repos {
		k.logger.Info("creating job for repository", "repo", repo)

		job := k.createJob(c, repo, retries, forkOrg)

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

func (k *KubernetesBackend) createJob(c *change.Change, repoURL string, retries int32, forkOrg string) *batchv1.Job {
	jobName := k.generateJobName(repoURL)
	image := c.Spec.Image
	if image == "" {
		image = DefaultImage
	}

	// Use retries from command-line argument
	backoffLimit := retries

	// Shared volume for repository
	sharedVolume := corev1.Volume{
		Name: "workspace",
		VolumeSource: corev1.VolumeSource{
			EmptyDir: &corev1.EmptyDirVolumeSource{},
		},
	}

	workspaceMount := corev1.VolumeMount{
		Name:      "workspace",
		MountPath: "/workspace",
	}

	// Init container 1: Create/sync fork
	forkSetupContainer := corev1.Container{
		Name:            "fork-setup",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"sh", "-c", forkSetupScript},
		VolumeMounts:    []corev1.VolumeMount{workspaceMount},
		Env: []corev1.EnvVar{
			{
				Name:  "ORIGINAL_REPO_URL",
				Value: repoURL,
			},
			{
				Name:  "FORK_ORG",
				Value: forkOrg,
			},
		},
		EnvFrom: []corev1.EnvFromSource{
			{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "baca-credentials",
					},
				},
			},
		},
	}

	// Init container 2: Clone fork repository with fleet
	branch := c.Spec.Branch
	if branch == "" {
		branch = "main"
	}

	// Clone from the fork (URL stored by fork-setup container)
	gitCloneContainer := corev1.Container{
		Name:            "git-clone",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command: []string{
			"sh", "-c",
			fmt.Sprintf("FORK_URL=$(cat /workspace/fork-url.txt); fleet gitcloner --branch %s \"$FORK_URL\" /workspace/repo", branch),
		},
		VolumeMounts: []corev1.VolumeMount{workspaceMount},
		EnvFrom: []corev1.EnvFromSource{
			{
				SecretRef: &corev1.SecretEnvSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "baca-credentials",
					},
				},
			},
		},
	}

	// Build JSON config for execute command
	configJSON, err := json.Marshal(c.Spec)
	if err != nil {
		k.logger.Error("failed to marshal config to JSON", "error", err)
		// Fallback to empty but this shouldn't happen
		configJSON = []byte("{}")
	}

	// Main container: Run baca execute with agent
	container := corev1.Container{
		Name:            "runner",
		Image:           image,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Command:         []string{"bash", "-c", jobScript},
		VolumeMounts:    []corev1.VolumeMount{workspaceMount},
		Env: []corev1.EnvVar{
			{
				Name:  "CONFIG",
				Value: string(configJSON),
			},
			{
				Name:  "REPO_URL",
				Value: repoURL,
			},
			{
				Name:  "ORIGINAL_REPO_URL",
				Value: repoURL,
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
						Name: "baca-credentials",
					},
				},
			},
		},
	}

	podSpec := corev1.PodSpec{
		RestartPolicy:  corev1.RestartPolicyNever,
		InitContainers: []corev1.Container{forkSetupContainer, gitCloneContainer},
		Containers:     []corev1.Container{container},
		Volumes:        []corev1.Volume{sharedVolume},
	}

	// Mount gemini OAuth files if using gemini-cli
	if c.Spec.Agent == "gemini-cli" {
		// Add gemini OAuth volume mount to the container
		container.VolumeMounts = append(container.VolumeMounts, corev1.VolumeMount{
			Name:      "gemini-oauth",
			MountPath: "/root/.gemini",
			ReadOnly:  true,
		})
		podSpec.Containers[0] = container

		// Add gemini OAuth volume to pod volumes
		podSpec.Volumes = append(podSpec.Volumes, corev1.Volume{
			Name: "gemini-oauth",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: "baca-credentials",
					Items: []corev1.KeyToPath{
						{Key: "GEMINI_oauth_creds.json", Path: "oauth_creds.json", Mode: int32Ptr(0600)},
						{Key: "GEMINI_google_accounts.json", Path: "google_accounts.json", Mode: int32Ptr(0600)},
						{Key: "GEMINI_installation_id", Path: "installation_id", Mode: int32Ptr(0600)},
						{Key: "GEMINI_settings.json", Path: "settings.json", Mode: int32Ptr(0600)},
					},
					Optional: boolPtr(true), // Optional in case using API key instead
				},
			},
		})
	}

	job := &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      jobName,
			Namespace: k.namespace,
			Labels: map[string]string{
				"app":                          "background-automated-code-agent",
				"app.kubernetes.io/name":       "baca",
				"app.kubernetes.io/component":  "job",
				"app.kubernetes.io/managed-by": "baca-cli",
				"repo":                         k.sanitizeLabel(repoURL),
			},
		},
		Spec: batchv1.JobSpec{
			// Automatically clean up jobs after completion
			TTLSecondsAfterFinished: int32Ptr(300),          // Clean up after 5 minutes
			BackoffLimit:            int32Ptr(backoffLimit), // Configurable retries (default: 0)
			Template: corev1.PodTemplateSpec{
				Spec: podSpec,
			},
		},
	}

	return job
}

func (k *KubernetesBackend) generateJobName(repoURL string) string {
	u, err := url.Parse(repoURL)
	if err != nil || u.Scheme == "" {
		return fmt.Sprintf("baca-job-%s", generateRandomSuffix())
	}

	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.ToLower(path)

	// Calculate max length: 63 (k8s limit) - len("baca-") - len("-") - 8 (suffix)
	const maxNameLen = 63
	const prefix = "baca-"
	const suffixLen = 8                                    // hex string length
	maxPathLen := maxNameLen - len(prefix) - 1 - suffixLen // -1 for hyphen before suffix

	if len(path) > maxPathLen {
		path = path[:maxPathLen]
	}

	return fmt.Sprintf("baca-%s-%s", path, generateRandomSuffix())
}

// generateRandomSuffix creates a short random string for job name uniqueness
func generateRandomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp if random fails
		return fmt.Sprintf("%x", time.Now().UnixNano()&0xffffffff)
	}
	return hex.EncodeToString(b)
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
	loggedPods := make(map[string]bool) // Track which pods we've already logged

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

				// When job completes or fails, print pod logs
				if (status == "Complete" || status == "Failed") && !loggedPods[jobName] {
					k.printPodLogs(ctx, jobName)
					loggedPods[jobName] = true
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

func (k *KubernetesBackend) printPodLogs(ctx context.Context, jobName string) {
	// Find the pod for this job
	podList := &corev1.PodList{}
	err := k.client.List(ctx, podList, client.InNamespace(k.namespace), client.MatchingLabels{
		"job-name": jobName,
	})
	if err != nil {
		k.logger.Error("failed to list pods for job", "job", jobName, "error", err)
		return
	}

	if len(podList.Items) == 0 {
		k.logger.Warn("no pods found for job", "job", jobName)
		return
	}

	pod := podList.Items[0]
	k.logger.Info("=== Pod logs for job ===", "job", jobName, "pod", pod.Name)

	// Get logs from all containers
	for _, container := range pod.Spec.InitContainers {
		k.printContainerLogs(ctx, pod.Name, container.Name, true)
	}
	for _, container := range pod.Spec.Containers {
		k.printContainerLogs(ctx, pod.Name, container.Name, false)
	}

	k.logger.Info("=== End of logs ===", "job", jobName)
}

func (k *KubernetesBackend) printContainerLogs(ctx context.Context, podName, containerName string, isInit bool) {
	containerType := "container"
	if isInit {
		containerType = "init-container"
	}

	req := k.clientset.CoreV1().Pods(k.namespace).GetLogs(podName, &corev1.PodLogOptions{
		Container: containerName,
	})

	logs, err := req.Stream(ctx)
	if err != nil {
		k.logger.Error("failed to get logs", "pod", podName, "container", containerName, "type", containerType, "error", err)
		return
	}
	defer logs.Close()

	k.logger.Info("--- Logs from "+containerType+" ---", "container", containerName)

	scanner := bufio.NewScanner(logs)
	for scanner.Scan() {
		// Print directly to stdout (not as structured log)
		fmt.Println(scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		k.logger.Error("error reading logs", "error", err)
	}
}
