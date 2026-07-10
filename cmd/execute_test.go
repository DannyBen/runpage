package cmd

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestExecuteBlockUsesConfiguredShellForConsole(t *testing.T) {
	t.Setenv("SHELL", "/bin/sh")
	message := executeBlock(context.Background(), 2, executableBlock{
		language: "console",
		command:  "printf '%s:%s' console-output \"$COLUMNS\"",
	}, 64)

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
	}, 80)

	if elapsed := time.Since(started); elapsed > 2*time.Second {
		t.Fatalf("cancellation took %v", elapsed)
	}
	if message.result.exitCode == 0 {
		t.Fatal("cancelled process reported success")
	}
	if strings.Contains(message.result.stdout, "sleep 10") {
		t.Fatalf("unexpected stdout: %q", message.result.stdout)
	}
}
