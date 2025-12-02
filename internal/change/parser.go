package change

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

func LoadFromFile(path string) (*Change, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read change file: %w", err)
	}

	var change Change
	if err := yaml.Unmarshal(data, &change); err != nil {
		return nil, fmt.Errorf("failed to parse change file: %w", err)
	}

	if err := validate(&change); err != nil {
		return nil, fmt.Errorf("invalid change definition: %w", err)
	}

	return &change, nil
}

func validate(c *Change) error {
	if c.Kind != "Change" {
		return fmt.Errorf("kind must be 'Change', got '%s'", c.Kind)
	}

	if c.Spec.Prompt == "" {
		return fmt.Errorf("spec.prompt is required")
	}

	if len(c.Spec.Repos) == 0 {
		return fmt.Errorf("spec.repos must contain at least one repository")
	}

	if c.Spec.Agent == "" {
		return fmt.Errorf("spec.agent is required")
	}

	return nil
}
