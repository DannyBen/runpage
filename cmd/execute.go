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

func executeBlock(ctx context.Context, index int, block executableBlock, columns int, workdir string) executionFinishedMsg {
	interpreter, arguments := interpreterCommand(block)

	command := exec.CommandContext(ctx, interpreter, arguments...)
	command.Dir = workdir
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
	result := executionResult{stdout: stdout.String(), stderr: stderr.String(), cancelled: ctx.Err() != nil}
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

func interpreterCommand(block executableBlock) (string, []string) {
	switch block.language {
	case "shell", "console":
		interpreter := os.Getenv("SHELL")
		if interpreter == "" {
			interpreter = "/bin/sh"
		}
		return interpreter, []string{"-c", block.command}
	case "python", "py":
		return "python3", []string{"-c", block.command}
	case "ruby", "rb":
		return "ruby", []string{"-e", block.command}
	default:
		return block.language, []string{"-c", block.command}
	}
}
