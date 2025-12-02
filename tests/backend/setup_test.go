package backend_test

import (
	"io"
	"log/slog"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/manno/background-coding-agent/internal/backend"
	"github.com/manno/background-coding-agent/tests/utils"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("Backend Setup", func() {
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
	})

	When("Setting up credentials", func() {
		It("creates secret with all credentials", func() {
			b, err := backend.New(cfg, namespace, logger)
			Expect(err).NotTo(HaveOccurred())

			credentials := map[string]string{
				"GITHUB_TOKEN":   "test-github-token",
				"GOOGLE_API_KEY": "test-google-key",
			}
			err = b.Setup(ctx, credentials)
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      "bca-credentials",
				Namespace: namespace,
			}, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret.Data["GITHUB_TOKEN"])).To(Equal("test-github-token"))
			Expect(string(secret.Data["GOOGLE_API_KEY"])).To(Equal("test-google-key"))
		})

		It("updates existing secret", func() {
			b, err := backend.New(cfg, namespace, logger)
			Expect(err).NotTo(HaveOccurred())

			credentials := map[string]string{
				"GITHUB_TOKEN": "initial-token",
			}
			err = b.Setup(ctx, credentials)
			Expect(err).NotTo(HaveOccurred())

			updatedCredentials := map[string]string{
				"GITHUB_TOKEN":   "updated-token",
				"COPILOT_TOKEN":  "new-copilot-token",
				"GEMINI_API_KEY": "new-gemini-key",
			}
			err = b.Setup(ctx, updatedCredentials)
			Expect(err).NotTo(HaveOccurred())

			secret := &corev1.Secret{}
			err = k8sClient.Get(ctx, client.ObjectKey{
				Name:      "bca-credentials",
				Namespace: namespace,
			}, secret)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(secret.Data["GITHUB_TOKEN"])).To(Equal("updated-token"))
			Expect(string(secret.Data["COPILOT_TOKEN"])).To(Equal("new-copilot-token"))
			Expect(string(secret.Data["GEMINI_API_KEY"])).To(Equal("new-gemini-key"))
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
})
