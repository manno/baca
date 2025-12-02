package change

type Change struct {
	Kind       string     `yaml:"kind"`
	APIVersion string     `yaml:"apiVersion"`
	Spec       ChangeSpec `yaml:"spec"`
}

type ChangeSpec struct {
	AgentsMD  string   `yaml:"agentsmd"`
	Resources []string `yaml:"resources"`
	Prompt    string   `yaml:"prompt"`
	Repos     []string `yaml:"repos"`
	Agent     string   `yaml:"agent"`
	Image     string   `yaml:"image,omitempty"`
	Branch    string   `yaml:"branch,omitempty"` // Git branch to checkout (default: "main")
}
