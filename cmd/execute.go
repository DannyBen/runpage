package cmd

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type executionFinishedMsg struct {
	index  int
	result executionResult
}

func executeBlock(ctx context.Context, index int, block executableBlock, columns int) executionFinishedMsg {
	interpreter := block.language
	if interpreter == "shell" || interpreter == "console" {
		interpreter = os.Getenv("SHELL")
		if interpreter == "" {
			interpreter = "/bin/sh"
		}
	}

	command := exec.CommandContext(ctx, interpreter, "-c", block.command)
	command.Env = append(os.Environ(), fmt.Sprintf("COLUMNS=%d", max(20, columns)))
	command.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	command.Cancel = func() error {
		if command.Process == nil {
			return os.ErrProcessDone
		}
		return syscall.Kill(-command.Process.Pid, syscall.SIGKILL)
	}
	command.WaitDelay = time.Second

	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	result := executionResult{stdout: stdout.String(), stderr: stderr.String()}
	if err == nil {
		return executionFinishedMsg{index: index, result: result}
	}
	var exitError *exec.ExitError
	if errors.As(err, &exitError) {
		result.exitCode = exitError.ExitCode()
	} else {
		result.exitCode = -1
		if result.stderr == "" {
			result.stderr = err.Error() + "\n"
		}
	}
	return executionFinishedMsg{index: index, result: result}
}
