package cmd

import (
	"strings"
)

type executableBlock struct {
	language string
	line     int
	command  string
}

type executionResult struct {
	exitCode int
	stdout   string
	stderr   string
}

func executableBlocks(markdown string) []executableBlock {
	lines := strings.Split(markdown, "\n")
	blocks := make([]executableBlock, 0)

	for index := 0; index < len(lines); index++ {
		marker, info, ok := openingFence(lines[index])
		if !ok {
			continue
		}
		language, executable := executableLanguage(info)
		start := index
		for index++; index < len(lines); index++ {
			if closingFence(lines[index], marker) {
				break
			}
		}
		if executable {
			blocks = append(blocks, executableBlock{
				language: language,
				line:     start,
				command:  strings.Join(lines[start+1:index], "\n"),
			})
		}
	}

	return blocks
}

func openingFence(line string) (marker, info string, ok bool) {
	trimmed := strings.TrimLeft(line, " \t")
	if len(line)-len(trimmed) > 3 || len(trimmed) < 3 {
		return "", "", false
	}
	fence := trimmed[0]
	if fence != '`' && fence != '~' {
		return "", "", false
	}
	length := 0
	for length < len(trimmed) && trimmed[length] == fence {
		length++
	}
	if length < 3 {
		return "", "", false
	}
	return trimmed[:length], strings.TrimSpace(trimmed[length:]), true
}

func closingFence(line, marker string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < len(marker) || trimmed[0] != marker[0] {
		return false
	}
	for index := 0; index < len(trimmed); index++ {
		if trimmed[index] != marker[0] {
			return false
		}
	}
	return len(trimmed) >= len(marker)
}

func executableLanguage(info string) (string, bool) {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return "", false
	}
	language := strings.ToLower(fields[0])
	for _, field := range fields[1:] {
		if field == ":noop" || field == "noop" {
			return language, false
		}
	}
	switch language {
	case "bash", "sh", "shell", "zsh", "console":
		return language, true
	default:
		return language, false
	}
}
