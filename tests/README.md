# Integration Tests

This directory contains integration tests using **Ginkgo** and **envtest** to test against a real Kubernetes API server.

## Structure

```
tests/
├── utils/              # Shared test utilities
│   ├── envtest.go     # envtest setup and configuration
│   ├── kubeconfig.go  # Kubeconfig generation
│   └── namespace.go   # Test namespace helpers
└── backend/            # Backend integration tests
    ├── suite_test.go  # Ginkgo suite setup
    └── backend_test.go # Backend test specs
```

## Prerequisites

### 1. Install Required Tools

```bash
# Ginkgo test framework
go install github.com/onsi/ginkgo/v2/ginkgo@latest

# envtest binaries manager
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

# Code formatting (recommended)
go install golang.org/x/tools/cmd/goimports@latest
```

### 2. Setup envtest Binaries

Download Kubernetes API server binaries:

```bash
setup-envtest use 1.34.1
```

### 3. Set KUBEBUILDER_ASSETS

**Required**: Set the `KUBEBUILDER_ASSETS` environment variable. Add to your shell config (`.bashrc`, `.zshrc`, or `.envrc`):

```bash
export KUBEBUILDER_ASSETS=$(setup-envtest use -p path 1.34.1)
```

Verify:
```bash
echo $KUBEBUILDER_ASSETS
# Should output: /path/to/envtest/k8s/1.34.1-<os>-<arch>
```

## Running Tests

### Integration Tests (Ginkgo)

Run backend integration tests:
```bash
ginkgo -v ./tests/backend/
```

Run all integration tests:
```bash
ginkgo -v ./tests/...
```

Run with coverage:
```bash
ginkgo -v --cover ./tests/...
```

### Standard Go Test

You can also use `go test`:
```bash
go test -v ./tests/backend/...
```

### Unit Tests

Unit tests in `internal/` packages (no K8s cluster needed):
```bash
go test ./internal/...
```

## Environment Variables

Configure test behavior with environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `KUBEBUILDER_ASSETS` | Path to envtest binaries | **Required** (unless using existing cluster) |
| `CI_USE_EXISTING_CLUSTER` | Use existing cluster instead of envtest | `false` |
| `CI_SILENCE_CTRL` | Silence controller-runtime logs | `false` |
| `CI_KUBECONFIG` | Write kubeconfig to this path | (none) |
| `SKIP_CLEANUP` | Skip cleanup after tests (debugging) | `false` |

### Using envtest (Default)

Run tests with ephemeral Kubernetes API server:
```bash
# Requires KUBEBUILDER_ASSETS to be set
export KUBEBUILDER_ASSETS=$(setup-envtest use -p path 1.34.1)
ginkgo -v ./tests/...
```

### Using k3d Cluster

Run tests against existing k3d cluster:
```bash
# Set CI_USE_EXISTING_CLUSTER to use your current cluster
export CI_USE_EXISTING_CLUSTER=true
ginkgo -v ./tests/...
```

**Note**: When `CI_USE_EXISTING_CLUSTER=true`, tests use your current kubeconfig context (e.g., k3d cluster). This is useful for:
- Debugging with real cluster state
- Testing with actual container images
- Avoiding envtest binary downloads

Example with debugging:
```bash
SKIP_CLEANUP=true CI_USE_EXISTING_CLUSTER=true ginkgo -v ./tests/backend/
```

## Writing Integration Tests

### Suite Structure

Each test package follows this pattern:

1. **`suite_test.go`** - Suite setup with BeforeSuite/AfterSuite
   - Initialize envtest environment
   - Start/stop test Kubernetes API server
   - Set up shared test resources

2. **`*_test.go`** - Individual test specs using Ginkgo BDD style
   - Use `Describe`, `Context`, `It` for test organization
   - Use `BeforeEach`/`AfterEach` for test isolation
   - Use `DeferCleanup` for resource cleanup

### Example Suite Setup

```go
// suite_test.go
package mypackage_test

import (
    "testing"
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/manno/background-coding-agent/tests/utils"
)

var testEnv *envtest.Environment

func TestMyPackage(t *testing.T) {
    RegisterFailHandler(Fail)
    RunSpecs(t, "My Package Suite")
}

var _ = BeforeSuite(func() {
    testEnv = utils.NewEnvTest()
    cfg, err := utils.StartTestEnv(testEnv)
    Expect(err).NotTo(HaveOccurred())
})

var _ = AfterSuite(func() {
    _ = testEnv.Stop()
})
```

### Example Test Spec

```go
// myfeature_test.go
package mypackage_test

import (
    . "github.com/onsi/ginkgo/v2"
    . "github.com/onsi/gomega"
    "github.com/manno/background-coding-agent/tests/utils"
    corev1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("My Feature", func() {
    var namespace string

    BeforeEach(func() {
        // Create unique test namespace
        namespace, _ = utils.NewNamespaceName()
        Expect(k8sClient.Create(ctx, &corev1.Namespace{
            ObjectMeta: metav1.ObjectMeta{Name: namespace},
        })).To(Succeed())

        DeferCleanup(func() {
            k8sClient.Delete(ctx, &corev1.Namespace{
                ObjectMeta: metav1.ObjectMeta{Name: namespace},
            })
        })
    })

    It("creates resources successfully", func() {
        obj := &corev1.ConfigMap{
            ObjectMeta: metav1.ObjectMeta{
                Name: "test", Namespace: namespace,
            },
        }
        Expect(k8sClient.Create(ctx, obj)).To(Succeed())
    })
})
```

## Test Guidelines

### Integration vs Unit Tests

| Type | Location | Infrastructure | Speed | Command |
|------|----------|---------------|-------|---------|
| **Unit** | `internal/*/` | None | Fast | `go test` |
| **Integration** | `tests/*/` | envtest K8s | Slower | `ginkgo` |

### Best Practices

- **Unit tests**: Test business logic without infrastructure
- **Integration tests**: Test Kubernetes interactions with envtest
- **Isolation**: Each test gets a unique namespace
- **Cleanup**: Use `DeferCleanup` for reliable resource cleanup
- **Timeouts**: Default 30s timeout, 3s polling (configured in `utils/envtest.go`)
- **Logging**: Tests output to `GinkgoWriter` for clean output

## Troubleshooting

### KUBEBUILDER_ASSETS not set
```
Error: unable to start control plane: unable to find binaries
```
**Solution**: Export `KUBEBUILDER_ASSETS` as shown in Prerequisites.

### Tests hanging
**Solution**: Check `SKIP_CLEANUP=true` wasn't left enabled.

### API server won't start
**Solution**: Ensure no port conflicts. envtest picks random ports automatically.

## References

- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)
- [envtest Documentation](https://book.kubebuilder.io/reference/envtest.html)
