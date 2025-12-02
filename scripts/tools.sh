#!/bin/bash
set -e
go install golang.org/x/tools/cmd/goimports@latest
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
