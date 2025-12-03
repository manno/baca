package backend

import (
	"strings"
	"testing"
)

func TestGenerateJobName(t *testing.T) {
	k := &KubernetesBackend{}

	tests := []struct {
		name      string
		repoURL   string
		wantCheck func(string) error
	}{
		{
			name:    "simple repository",
			repoURL: "https://github.com/manno/fleet",
			wantCheck: func(jobName string) error {
				if !strings.HasPrefix(jobName, "bca-manno-fleet-") {
					t.Errorf("expected job name to start with 'bca-manno-fleet-', got %s", jobName)
				}
				return nil
			},
		},
		{
			name:    "repository with .git suffix",
			repoURL: "https://github.com/kubernetes/kubernetes.git",
			wantCheck: func(jobName string) error {
				if !strings.HasPrefix(jobName, "bca-kubernetes-kubernetes-") {
					t.Errorf("expected job name to start with 'bca-kubernetes-kubernetes-', got %s", jobName)
				}
				return nil
			},
		},
		{
			name:    "very long repository name",
			repoURL: "https://github.com/organization-name/very-long-repository-name-that-exceeds-kubernetes-limits",
			wantCheck: func(jobName string) error {
				if len(jobName) > 63 {
					t.Errorf("job name exceeds Kubernetes limit: length=%d, name=%s", len(jobName), jobName)
				}
				return nil
			},
		},
		{
			name:    "invalid URL (no scheme)",
			repoURL: "not-a-valid-url",
			wantCheck: func(jobName string) error {
				if !strings.HasPrefix(jobName, "bca-job-") {
					t.Errorf("expected fallback job name to start with 'bca-job-', got %s", jobName)
				}
				return nil
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			jobName := k.generateJobName(tt.repoURL)

			// Check length is within Kubernetes limits
			if len(jobName) > 63 {
				t.Errorf("job name exceeds Kubernetes 63 character limit: length=%d, name=%s", len(jobName), jobName)
			}

			// Check it's lowercase and valid
			if jobName != strings.ToLower(jobName) {
				t.Errorf("job name should be lowercase, got %s", jobName)
			}

			// Check it starts with bca-
			if !strings.HasPrefix(jobName, "bca-") {
				t.Errorf("job name should start with 'bca-', got %s", jobName)
			}

			// Run custom check
			if tt.wantCheck != nil {
				if err := tt.wantCheck(jobName); err != nil {
					t.Error(err)
				}
			}
		})
	}
}

func TestGenerateJobNameUniqueness(t *testing.T) {
	k := &KubernetesBackend{}
	repoURL := "https://github.com/manno/fleet"

	names := make(map[string]bool)
	iterations := 100

	for i := 0; i < iterations; i++ {
		name := k.generateJobName(repoURL)
		if names[name] {
			t.Errorf("duplicate job name generated: %s", name)
		}
		names[name] = true
	}

	if len(names) != iterations {
		t.Errorf("expected %d unique names, got %d", iterations, len(names))
	}
}

func TestGenerateRandomSuffix(t *testing.T) {
	seen := make(map[string]bool)
	iterations := 1000

	for i := 0; i < iterations; i++ {
		suffix := generateRandomSuffix()

		// Check length (should be 8 hex characters)
		if len(suffix) != 8 {
			t.Errorf("expected suffix length 8, got %d: %s", len(suffix), suffix)
		}

		// Check it's valid hex
		for _, c := range suffix {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				t.Errorf("suffix contains non-hex character: %s", suffix)
			}
		}

		seen[suffix] = true
	}

	// Check we got mostly unique values (allow for small collision rate)
	uniqueRate := float64(len(seen)) / float64(iterations)
	if uniqueRate < 0.99 {
		t.Errorf("suffix collision rate too high: %.2f%% unique", uniqueRate*100)
	}
}
