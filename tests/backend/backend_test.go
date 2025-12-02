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
			b, err := backend.NewKubernetesBackend(cfg, namespace, logger)
			Expect(err).NotTo(HaveOccurred())
			Expect(b).NotTo(BeNil())
		})

		It("can call Setup", func() {
			b, err := backend.NewKubernetesBackend(cfg, namespace, logger)
			Expect(err).NotTo(HaveOccurred())

			err = b.Setup(ctx)
			Expect(err).NotTo(HaveOccurred())
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
