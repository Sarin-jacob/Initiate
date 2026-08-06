package internal

import (
	"bytes"
	"os/exec"
	"text/template"
)

// AgentConfig represents the parsed config.yaml
type AgentConfig struct {
	Server struct {
		URL            string `yaml:"url"`
		CertPin        string `yaml:"cert_pin"` // e.g. "sha256:abc123def..."
		PrivateKeyPath string `yaml:"private_key_path"`
	} `yaml:"server"`
	
	Executor map[string]CommandConfig `yaml:"executor"`
}

// CommandConfig maps an event (like PROVISION) to an OS command
type CommandConfig struct {
	Command string   `yaml:"command"` // e.g. "/usr/sbin/useradd"
	Args    []string `yaml:"args"`    // e.g. ["-m", "-s", "/bin/bash", "{{.Username}}"]
}

// ExecuteTask safely injects variables into arguments and runs the local OS command
func ExecuteTask(cmdConf CommandConfig, payload map[string]interface{}) (string, error) {
	var parsedArgs []string

	// Safely parse each argument template with the incoming JSON payload
	for _, argTpl := range cmdConf.Args {
		t, err := template.New("arg").Parse(argTpl)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := t.Execute(&buf, payload); err != nil {
			return "", err
		}
		parsedArgs = append(parsedArgs, buf.String())
	}

	// Execute natively on the host Linux OS
	cmd := exec.Command(cmdConf.Command, parsedArgs...)
	out, err := cmd.CombinedOutput()
	
	return string(out), err
}