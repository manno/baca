package utils

import (
	"os"
	"time"

	"github.com/go-logr/logr"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

const (
	Timeout         = 30 * time.Second
	PollingInterval = 3 * time.Second
)

func init() {
	gomega.SetDefaultEventuallyTimeout(Timeout)
	gomega.SetDefaultEventuallyPollingInterval(PollingInterval)
}

func NewEnvTest() *envtest.Environment {
	if os.Getenv("CI_SILENCE_CTRL") != "" {
		ctrl.SetLogger(logr.New(log.NullLogSink{}))
	} else {
		ctrl.SetLogger(zap.New(zap.WriteTo(ginkgo.GinkgoWriter), zap.UseDevMode(true)))
	}

	existing := os.Getenv("CI_USE_EXISTING_CLUSTER") == "true"
	return &envtest.Environment{
		UseExistingCluster: &existing,
	}
}

func StartTestEnv(testEnv *envtest.Environment) (*rest.Config, error) {
	cfg, err := testEnv.Start()
	if err != nil {
		return nil, err
	}

	if config := os.Getenv("CI_KUBECONFIG"); config != "" {
		err = WriteKubeConfig(cfg, config)
	}

	return cfg, err
}
