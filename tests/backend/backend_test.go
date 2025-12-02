package backend_test

import (
	"io"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/manno/background-coding-agent/internal/backend"
	"github.com/manno/background-coding-agent/internal/change"
	"github.com/manno/background-coding-agent/tests/utils"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Backend Integration", func() {
	var logger *slog.Logger

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
	})

	When("Creating KubernetesBackend", func() {
		It("successfully creates backend with config", func() {
			b, err := backend.New(cfg, namespace, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(b).NotTo(BeNil())
		})

		It("can call Setup", func() {
			b, err := backend.New(cfg, namespace, logger)
			Expect(err).NotTo(HaveOccurred())

			err = b.Setup(ctx, "test-github-token")
			Expect(err).NotTo(HaveOccurred())

			// Verify secret was created
			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      "bca-credentials",
				Namespace: namespace,
			}, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret.Data["GITHUB_TOKEN"])).To(Equal("test-github-token"))
		})
	})

	When("Using GetConfig", func() {
		It("can get config from kubeconfig file", func() {
			cfg2, err := backend.GetConfig(kubeconfigPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(cfg2).NotTo(BeNil())
			Expect(cfg2.Host).To(Equal(cfg.Host))
		})
	})

	When("ApplyChange", func() {
		var b *backend.KubernetesBackend

		BeforeEach(func() {
			var err error
			b, err = backend.New(cfg, namespace, logger)
			Expect(err).NotTo(HaveOccurred())

			// Setup credentials first
			err = b.Setup(ctx, "test-github-token")
			Expect(err).NotTo(HaveOccurred())
		})

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

			err := b.ApplyChange(ctx, ch, false)
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(2))

			for _, job := range jobList.Items {
				Expect(job.Spec.Template.Spec.Containers).To(HaveLen(1))
				Expect(job.Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/example/runner:latest"))
				Expect(job.Labels["app"]).To(Equal("background-coding-agent"))
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

			err := b.ApplyChange(ctx, ch, false)
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))

			container := jobList.Items[0].Spec.Template.Spec.Containers[0]
			Expect(container.EnvFrom).To(HaveLen(1))
			Expect(container.EnvFrom[0].SecretRef.Name).To(Equal("bca-credentials"))
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

			err := b.ApplyChange(ctx, ch, false)
			Expect(err).NotTo(HaveOccurred())

			jobList := &batchv1.JobList{}
			err = k8sClient.List(ctx, jobList, client.InNamespace(namespace))
			Expect(err).NotTo(HaveOccurred())
			Expect(jobList.Items).To(HaveLen(1))
			Expect(jobList.Items[0].Spec.Template.Spec.Containers[0].Image).To(Equal("ghcr.io/manno/background-coding-agent:latest"))
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

			err := b.ApplyChange(ctx, ch, false)
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
	})
})
