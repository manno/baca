package backend_test

import (
	"io"
	"log/slog"
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/manno/baca/internal/backend/k8s"
	"github.com/manno/baca/internal/change"
	"github.com/manno/baca/tests/utils"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Backend Apply", func() {
	var logger *slog.Logger
	var b *k8s.KubernetesBackend

	BeforeEach(func() {
		logger = slog.New(slog.NewTextHandler(io.Discard, nil))

		var err error
		namespace, err = utils.NewNamespaceName()
		Expect(err).ToNot(HaveOccurred())
		Expect(k8sClient.Create(ctx, &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: namespace,
			},
		})).ToNot(HaveOccurred())

		DeferCleanup(func() {
			Expect(k8sClient.Delete(ctx, &corev1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: namespace,
				},
			})).ToNot(HaveOccurred())
		})

		b, err = k8s.New(cfg, namespace, logger)
		Expect(err).NotTo(HaveOccurred())

		// Setup credentials - use real tokens from environment if available, otherwise use test tokens
		githubToken := os.Getenv("GITHUB_TOKEN")
		if githubToken == "" {
			githubToken = os.Getenv("COPILOT_TOKEN")
		}
		if githubToken == "" {
			githubToken = "test-github-token" //nolint:gosec // G101: Test credential, not production
		}

		credentials := map[string]string{
			"GITHUB_TOKEN": githubToken,
		}

		// Add optional credentials if available
		if copilotToken := os.Getenv("COPILOT_TOKEN"); copilotToken != "" {
			credentials["COPILOT_TOKEN"] = copilotToken
		}
		if geminiKey := os.Getenv("GEMINI_API_KEY"); geminiKey != "" {
			credentials["GEMINI_API_KEY"] = geminiKey
		}

		err = b.Setup(ctx, credentials)
		Expect(err).NotTo(HaveOccurred())
	})

	When("ApplyChange", func() {
		It("creates a job for each repository", func() {
			ch := &change.Change{
				APIVersion: "v1",
				Kind:       "Change",
				Spec: change.ChangeSpec{
					AgentsMD: "https://example.com/agents.md",
					Resources: []string{
						"https://example.com/docs/guide.md",
					},
					Prompt: "Add error handling",
					Repos: []string{
						"https://github.com/example/repo1",
						"https://github.com/example/repo2",
					},
					Agent: "gemini-cli",
					Image: "ghcr.io/example/runner:latest",
				},
			}

			err := b.ApplyChange(ctx, ch, false, 0, "")
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(2))

			for _, job := range jobList.Items {
				Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
				Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/example/runner:latest"))
				Expect(job.Labels["app"]).To(Equal("background-automated-code-agent"))
			}
		})

		It("mounts credentials secret in jobs", func() {
			ch := &change.Change{
				APIVersion: "v1",
				Kind:       "Change",
				Spec: change.ChangeSpec{
					AgentsMD: "https://example.com/agents.md",
					Prompt:   "Add tests",
					Repos: []string{
						"https://github.com/example/repo1",
					},
					Agent: "copilot-cli",
				},
			}

			err := b.ApplyChange(ctx, ch, false, 0, "")
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))

			container := jobList.Items[0].Spec.Template.Spec.Containers[0]
			Expect(container.EnvFrom).To(HaveLen(1))
			Expect(container.EnvFrom[0].SecretRef.Name).To(Equal("baca-credentials"))
		})

		It("uses default image when not specified", func() {
			ch := &change.Change{
				APIVersion: "v1",
				Kind:       "Change",
				Spec: change.ChangeSpec{
					AgentsMD: "https://example.com/agents.md",
					Prompt:   "Add tests",
					Repos: []string{
						"https://github.com/example/repo1",
					},
					Agent: "copilot-cli",
				},
			}

			err := b.ApplyChange(ctx, ch, false, 0, "")
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))
			Expect(jobList.Items[0].Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/manno/baca-runner:latest"))
		})

		It("can get job status", func() {
			ch := &change.Change{
				APIVersion: "v1",
				Kind:       "Change",
				Spec: change.ChangeSpec{
					AgentsMD: "https://example.com/agents.md",
					Prompt:   "Add tests",
					Repos: []string{
						"https://github.com/example/repo1",
					},
					Agent: "copilot-cli",
				},
			}

			err := b.ApplyChange(ctx, ch, false, 0, "")
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))

			jobName := jobList.Items[0].Name
			status, err := b.GetJobStatus(ctx, jobName)
			Expect(err).NotTo(HaveOccurred())
			Expect(status).To(BeElementOf("Pending", "Running"))
		})

		It("sets FORK_ORG environment variable when fork-org is specified", func() {
			ch := &change.Change{
				APIVersion: "v1",
				Kind:       "Change",
				Spec: change.ChangeSpec{
					Prompt: "Add tests",
					Repos: []string{
						"https://github.com/example/repo1",
					},
					Agent: "copilot-cli",
				},
			}

			err := b.ApplyChange(ctx, ch, false, 0, "test-org")
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))

			// Check fork-setup init container has FORK_ORG env var
			job := jobList.Items[0]
			Expect(job.Spec.Template.Spec.InitContainers).To(HaveLen(2))
			forkSetupContainer := job.Spec.Template.Spec.InitContainers[0]
			Expect(forkSetupContainer.Name).To(Equal("fork-setup"))

			var foundForkOrg bool
			var forkOrgValue string
			for _, env := range forkSetupContainer.Env {
				if env.Name == "FORK_ORG" {
					foundForkOrg = true
					forkOrgValue = env.Value
					break
				}
			}
			Expect(foundForkOrg).To(BeTrue(), "FORK_ORG environment variable should be set")
			Expect(forkOrgValue).To(Equal("test-org"))
		})

		It("sets empty FORK_ORG when fork-org is not specified", func() {
			ch := &change.Change{
				APIVersion: "v1",
				Kind:       "Change",
				Spec: change.ChangeSpec{
					Prompt: "Add tests",
					Repos: []string{
						"https://github.com/example/repo1",
					},
					Agent: "copilot-cli",
				},
			}

			err := b.ApplyChange(ctx, ch, false, 0, "")
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))

			// Check fork-setup init container has empty FORK_ORG env var
			job := jobList.Items[0]
			forkSetupContainer := job.Spec.Template.Spec.InitContainers[0]

			var foundForkOrg bool
			var forkOrgValue string
			for _, env := range forkSetupContainer.Env {
				if env.Name == "FORK_ORG" {
					foundForkOrg = true
					forkOrgValue = env.Value
					break
				}
			}
			Expect(foundForkOrg).To(BeTrue(), "FORK_ORG environment variable should be set")
			Expect(forkOrgValue).To(Equal(""))
		})
	})
})
