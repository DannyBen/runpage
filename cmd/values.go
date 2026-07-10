package cmd

import (
	"fmt"
	"strings"
)

type documentValues map[string]string

func parseDocumentValues(arguments []string) (documentValues, error) {
	values := make(documentValues, len(arguments))
	for _, argument := range arguments {
		key, value, ok := strings.Cut(argument, ":")
		if !ok || !validValueKey(key) {
			return nil, fmt.Errorf("invalid document value %q (expected KEY:VALUE)", argument)
		}
		if _, exists := values[key]; exists {
			return nil, fmt.Errorf("duplicate document value: %s", key)
		}
		values[key] = value
	}
	return values, nil
}

func validValueKey(key string) bool {
	if key == "" || !valueKeyStart(key[0]) {
		return false
	}
	for index := 1; index < len(key); index++ {
		if !valueKeyPart(key[index]) {
			return false
		}
	}
	return true
}

func valueKeyStart(character byte) bool {
	return character == '_' || character >= 'a' && character <= 'z' || character >= 'A' && character <= 'Z'
}

func valueKeyPart(character byte) bool {
	return valueKeyStart(character) || character >= '0' && character <= '9'
}

func injectDocumentValues(markdown []byte, values documentValues) []byte {
	if len(values) == 0 {
		return markdown
	}

	source := string(markdown)
	var result strings.Builder
	result.Grow(len(source))
	for index := 0; index < len(source); {
		if strings.HasPrefix(source[index:], "{{") {
			if end := strings.Index(source[index+2:], "}}"); end >= 0 {
				end += index + 2
				key := strings.TrimSpace(source[index+2 : end])
				if value, ok := values[key]; ok {
					result.WriteString(value)
					index = end + 2
					continue
				}
			}
		}
		if source[index] == '$' {
			end := index + 1
			if end < len(source) && valueKeyStart(source[end]) {
				end++
				for end < len(source) && valueKeyPart(source[end]) {
					end++
				}
				if value, ok := values[source[index+1:end]]; ok {
					result.WriteString(value)
					index = end
					continue
				}
			}
		}
		result.WriteByte(source[index])
		index++
	}
	return []byte(result.String())
}
