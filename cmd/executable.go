package cmd

import (
	"strings"
)

type executableBlock struct {
	language string
	line     int
	endLine  int
	command  string
	tags     []string
}

type executionResult struct {
	exitCode  int
	stdout    string
	stderr    string
	cancelled bool
}

func executableBlocks(markdown string) []executableBlock {
	lines := strings.Split(markdown, "\n")
	blocks := make([]executableBlock, 0)

	for index := 0; index < len(lines); index++ {
		marker, info, ok := openingFence(lines[index])
		if !ok {
			continue
		}
		language, _, tags, executable := fenceMetadata(info)
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
				endLine:  index,
				command:  strings.Join(lines[start+1:index], "\n"),
				tags:     tags,
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

func fenceMetadata(info string) (language, label string, tags []string, executable bool) {
	fields := strings.Fields(info)
	if len(fields) == 0 {
		return "", "", nil, false
	}
	language = strings.ToLower(fields[0])
	noop := false
	labels := make([]string, 0, len(fields)-1)
	for _, field := range fields[1:] {
		if field == ":noop" || field == "noop" {
			noop = true
			continue
		}
		if strings.HasPrefix(field, ":") {
			tags = append(tags, strings.TrimPrefix(field, ":"))
			continue
		}
		labels = append(labels, field)
	}
	label = strings.Join(labels, " ")
	switch language {
	case "bash", "sh", "shell", "zsh", "console", "python", "py", "ruby", "rb":
		return language, label, tags, !noop
	default:
		return language, label, tags, false
	}
}

func (block executableBlock) hasTag(tag string) bool {
	for _, candidate := range block.tags {
		if candidate == tag {
			return true
		}
	}
	return false
}
