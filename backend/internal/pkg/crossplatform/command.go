package crossplatform

import (
	"context"
	"os/exec"
	"time"
)

// CommandResult represents the result of a command execution
type CommandResult struct {
	Command  string
	Output   string
	ExitCode int
	Duration time.Duration
	Error    error
}

// ExecuteCommand executes a command with context
func ExecuteCommand(ctx context.Context, command string, workingDir string, env []string) (*CommandResult, error) {
	result := &CommandResult{
		Command: command,
	}

	shell := GetShell()
	shellArgs := GetShellArgs()
	args := append(shellArgs, command)
	
	cmd := exec.CommandContext(ctx, shell, args...)
	
	if workingDir != "" {
		cmd.Dir = workingDir
	}
	
	if env != nil {
		cmd.Env = env
	}

	start := time.Now()
	output, err := cmd.CombinedOutput()
	result.Duration = time.Since(start)
	result.Output = string(output)

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		result.Error = err
		return result, err
	}

	result.ExitCode = 0
	return result, nil
}

// BuildEnvironment builds the command environment variables
func BuildEnvironment(workingDir string) []string {
	env := []string{
		"PATH=" + GetDefaultPath(),
		"HOME=" + workingDir,
		"PWD=" + workingDir,
	}

	env = append(env, "NODE_ENV=development")
	env = append(env, "PYTHONUNBUFFERED=1")
	env = append(env, "GOPROXY=https://proxy.golang.org,direct")
	env = append(env, "CARGO_NET_OFFLINE=false")

	return env
}
