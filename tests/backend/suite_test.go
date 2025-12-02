package backend_test

import (
	"context"
	"os"
	"path"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/manno/background-coding-agent/tests/utils"

	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/manno/background-coding-agent/internal/backend"
)

var (
	ctx            context.Context
	cancel         context.CancelFunc
	cfg            *rest.Config
	testEnv        *envtest.Environment
	tmpdir         string
	kubeconfigPath string

	k8sClient client.Client
	namespace string
)

func TestBackend(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Backend Integration Suite")
}

var _ = BeforeSuite(func() {
	ctx, cancel = context.WithCancel(context.TODO())
	testEnv = utils.NewEnvTest()

	var err error
	cfg, err = utils.StartTestEnv(testEnv)
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	tmpdir, _ = os.MkdirTemp("", "bca-")
	kubeconfigPath = path.Join(tmpdir, "kubeconfig")
	err = utils.WriteKubeConfig(cfg, kubeconfigPath)
	Expect(err).NotTo(HaveOccurred())

	k8sClient, err = backend.NewClient(cfg)
	Expect(err).NotTo(HaveOccurred())
	Expect(k8sClient).NotTo(BeNil())
})

var _ = AfterSuite(func() {
	if os.Getenv("SKIP_CLEANUP") == "true" {
		return
	}
	os.RemoveAll(tmpdir)

	cancel()
	_ = testEnv.Stop()
})
