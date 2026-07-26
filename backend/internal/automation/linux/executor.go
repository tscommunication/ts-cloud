package linux

import (
	"bytes"
	"fmt"
	"io"
	"os/exec"
)

type CommandResult struct {
	Stdout string
	Stderr string
}

func Execute(command string, args ...string) (*CommandResult, error) {

	return ExecuteWithInput("", command, args...)
}

func ExecuteWithInput(input string, command string, args ...string) (*CommandResult, error) {

	cmd := exec.Command(command, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if input != "" {
		cmd.Stdin = io.NopCloser(bytes.NewBufferString(input))
	}

	if err := cmd.Run(); err != nil {

		return &CommandResult{
			Stdout: stdout.String(),
			Stderr: stderr.String(),
		}, fmt.Errorf("%w: %s", err, stderr.String())
	}

	return &CommandResult{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}, nil
}
