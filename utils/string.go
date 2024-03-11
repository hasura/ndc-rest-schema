package utils

import (
	"regexp"
	"strings"
)

var nonAlphaDigitRegex = regexp.MustCompile(`[^\w]+`)

// ToPascalCase convert a string to PascalCase
func ToPascalCase(input string) string {
	if input == "" {
		return input
	}
	input = nonAlphaDigitRegex.ReplaceAllString(input, "_")
	parts := strings.Split(input, "_")
	for i := range parts {
		if parts[i] == "" {
			continue
		}
		parts[i] = strings.ToUpper(parts[i][:1]) + parts[i][1:]
	}
	return strings.Join(parts, "")
}

// StringSliceToPascalCase convert a slice of string to PascalCase
func StringSliceToPascalCase(inputs []string) string {
	if len(inputs) == 0 {
		return ""
	}

	results := make([]string, len(inputs))
	for i, item := range inputs {
		results[i] = ToPascalCase(item)
	}
	return strings.Join(results, "")
}
