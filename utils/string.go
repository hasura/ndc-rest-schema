package utils

import (
	"regexp"
	"strings"
	"unicode"
)

var nonAlphaDigitRegex = regexp.MustCompile(`[^\w]+`)

// ToCamelCase convert a string to camelCase
func ToCamelCase(input string) string {
	pascalCase := ToPascalCase(input)
	if pascalCase == "" {
		return pascalCase
	}
	return strings.ToLower(pascalCase[:1]) + pascalCase[1:]
}

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

// ToSnakeCase converts string to snake_case
func ToSnakeCase(input string) string {
	var sb strings.Builder
	inputLen := len(input)
	for i := 0; i < inputLen; i++ {
		char := rune(input[i])
		if char == '_' || char == '-' {
			sb.WriteRune('_')
			continue
		}
		if unicode.IsDigit(char) || unicode.IsLower(char) {
			sb.WriteRune(char)
			continue
		}

		if unicode.IsUpper(char) {
			if i == 0 {
				sb.WriteRune(unicode.ToLower(char))
				continue
			}
			lastChar := rune(input[i-1])
			if unicode.IsDigit(lastChar) || unicode.IsLower(lastChar) {
				sb.WriteRune('_')
				sb.WriteRune(unicode.ToLower(char))
				continue
			}
			if i < inputLen-1 {
				nextChar := rune(input[i+1])
				if unicode.IsUpper(lastChar) && !unicode.IsUpper(nextChar) {
					sb.WriteRune('_')
					sb.WriteRune(unicode.ToLower(char))
					continue
				}
			}

			sb.WriteRune(unicode.ToLower(char))
		}
	}
	return sb.String()
}

// ToConstantCase converts string to CONSTANT_CASE
func ToConstantCase(input string) string {
	return strings.ToUpper(ToSnakeCase(input))
}
