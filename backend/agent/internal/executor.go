package internal

import (
	"bytes"
	"os/exec"
	"text/template"
)

// ExecuteTask safely injects variables into arguments and runs the local OS command
func ExecuteTask(actionConf ActionConfig, payload map[string]interface{}) (string, error) {
	var parsedArgs []string

	// Safely parse each argument template with the incoming JSON payload
	for _, argTpl := range actionConf.Args {
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

	// Execute natively on the host Linux OS (ignores shell aliases and injections)
	cmd := exec.Command(actionConf.Command, parsedArgs...)
	out, err := cmd.CombinedOutput()

	return string(out), err
}