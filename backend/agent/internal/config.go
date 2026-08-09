package internal

import (
	"os"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	URL            string `yaml:"url"`
	CertPin        string `yaml:"cert_pin"`
	PrivateKeyPath string `yaml:"private_key_path"`
}

// ActionConfig defines a single symmetrical action (e.g., 'create' or 'delete')
type ActionConfig struct {
	Description string   `yaml:"description"` // UI Metadata: What does this do?
	Variables   map[string]string `yaml:"variables"`   // UI Metadata: Expected JSON payload keys
	Command     string   `yaml:"command"`     // e.g., "/usr/sbin/useradd"
	Args        []string `yaml:"args"`        // e.g., ["-m", "{{.username}}"]
}

// ModuleConfig holds a map of lifecycle actions (create, delete, suspend, resume)
type ModuleConfig map[string]ActionConfig

// AgentConfig represents the entire config.yaml file
type AgentConfig struct {
	Server  ServerConfig            `yaml:"server"`
	Modules map[string]ModuleConfig `yaml:"modules"`
}

// LoadConfig securely reads and parses the YAML file
func LoadConfig(path string) (*AgentConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var config AgentConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &config, nil
}