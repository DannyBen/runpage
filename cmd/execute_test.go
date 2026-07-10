package cmd

import (
	"context"
	"os/exec"
	"strings"
	"testing"
	"time"
)

func TestExecuteBlockUsesConfiguredShellForConsole(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	message := executeBlock(context.Background(), 2, executableBlock{
		language: "console",
		command:  "printf '%s:%s' console-output \"$COLUMNS\"",
	}, 64, "")

	if message.index != 2 || message.result.exitCode != 0 {
		t.Fatalf("message = %#v", message)
	}
	if message.result.stdout != "console-output:64" {
		t.Fatalf("stdout = %q", message.result.stdout)
	}
}

func TestExecuteBlockCancellationStopsProcess(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	started := time.Now()

	message := executeBlock(ctx, 0, executableBlock{
		language: "sh",
		command:  "sleep 10",
	}, 80, "")

	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("cancellation took %v", elapsed)
	}
	if message.result.exitCode == 0 {
		t.Fatal("cancelled process reported success")
	}
	if !message.result.cancelled {
		t.Fatal("cancelled process was not marked cancelled")
	}
	if strings.Contains(message.result.stdout, "sleep 10") {
		t.Fatalf("unexpected stdout: %q", message.result.stdout)
	}
}

func TestExecuteBlockReportsMissingInterpreter(t *testing.T) {
	message := executeBlock(context.Background(), 0, executableBlock{
		language: "/definitely/not/a/shell",
		command:  "printf unreachable",
	}, 80, "")

	if message.result.exitCode != -1 {
		t.Fatalf("exitCode = %d, want -1", message.result.exitCode)
	}
	if !strings.Contains(message.result.stderr, "no such file or directory") {
		t.Fatalf("stderr = %q", message.result.stderr)
	}
}

func TestInterpreterCommand(t *testing.T) {
	tests := []struct {
		language string
		wantName string
		wantArgs string
	}{
		{language: "python", wantName: "python3", wantArgs: "-c|print('ok')"},
		{language: "py", wantName: "python3", wantArgs: "-c|print('ok')"},
		{language: "ruby", wantName: "ruby", wantArgs: "-e|print('ok')"},
		{language: "rb", wantName: "ruby", wantArgs: "-e|print('ok')"},
	}
	for _, test := range tests {
		t.Run(test.language, func(t *testing.T) {
			name, args := interpreterCommand(executableBlock{language: test.language, command: "print('ok')"})
			if name != test.wantName || strings.Join(args, "|") != test.wantArgs {
				t.Fatalf("command = %q %#v", name, args)
			}
		})
	}
}

func TestExecuteBlockUsesWorkdir(t *testing.T) {
	workdir := t.TempDir()
	message := executeBlock(context.Background(), 0, executableBlock{
		language: "sh",
		command:  "pwd",
	}, 80, workdir)

	if message.result.exitCode != 0 {
		t.Fatalf("result = %#v", message.result)
	}
	if got := strings.TrimSpace(message.result.stdout); got != workdir {
		t.Fatalf("pwd = %q, want %q", got, workdir)
	}
}

func TestExecuteBlockWithPythonAndRuby(t *testing.T) {
	tests := []struct {
		name        string
		interpreter string
		command     string
		want        string
	}{
		{name: "python", interpreter: "python3", command: "print('python-output')", want: "python-output\n"},
		{name: "ruby", interpreter: "ruby", command: "puts 'ruby-output'", want: "ruby-output\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := exec.LookPath(test.interpreter); err != nil {
				t.Skipf("%s is not installed", test.interpreter)
			}
			message := executeBlock(context.Background(), 0, executableBlock{
				language: test.name,
				command:  test.command,
			}, 80, "")
			if message.result.exitCode != 0 || message.result.stdout != test.want {
				t.Fatalf("result = %#v", message.result)
			}
		})
	}
}
