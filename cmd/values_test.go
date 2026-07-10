package cmd

import (
	"strings"
	"testing"
)

func TestParseDocumentValues(t *testing.T) {
	values, err := parseDocumentValues([]string{"version:1.0.0", "url:https://example.com:8443/release", "empty:"})
	if err != nil {
		t.Fatalf("parseDocumentValues returned error: %v", err)
	}
	if values["version"] != "1.0.0" || values["url"] != "https://example.com:8443/release" || values["empty"] != "" {
		t.Fatalf("values = %#v", values)
	}
}

func TestParseDocumentValuesRejectsInvalidArguments(t *testing.T) {
	for _, arguments := range [][]string{
		{"version"},
		{"1version:value"},
		{"release-version:value"},
		{"version:one", "version:two"},
	} {
		if _, err := parseDocumentValues(arguments); err == nil {
			t.Fatalf("arguments %#v did not return an error", arguments)
		}
	}
}

func TestInjectDocumentValues(t *testing.T) {
	markdown := []byte(strings.Join([]string{
		"Release $version",
		"Release {{ version }}",
		"Release {{version}}",
		"Name $version_name",
		"Unknown $missing {{ missing }}",
		"Literal $payload",
	}, "\n"))
	values := documentValues{
		"version":      "1.0.0",
		"version_name": "stable",
		"payload":      "$version {{ version }}",
	}

	got := string(injectDocumentValues(markdown, values))
	for _, want := range []string{
		"Release 1.0.0",
		"Name stable",
		"Unknown $missing {{ missing }}",
		"Literal $version {{ version }}",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("injected document missing %q:\n%s", want, got)
		}
	}
}

func TestInjectedValuesReachExecutableBlocks(t *testing.T) {
	markdown := injectDocumentValues(
		[]byte("```sh\nprintf '%s' '$version'\n```\n"),
		documentValues{"version": "1.0.0"},
	)
	blocks := executableBlocks(string(markdown))
	if len(blocks) != 1 || blocks[0].command != "printf '%s' '1.0.0'" {
		t.Fatalf("blocks = %#v", blocks)
	}
}
