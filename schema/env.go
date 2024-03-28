package schema

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var envVariableRegex = regexp.MustCompile(`{{([A-Z0-9_]+)(:-([^}]*))?}}`)

// EnvTemplate represents an environment variable template
type EnvTemplate struct {
	Name         string
	DefaultValue *string
}

// NewEnvTemplate creates an EnvTemplate without default value
func NewEnvTemplate(name string) *EnvTemplate {
	return &EnvTemplate{
		Name: name,
	}
}

// NewEnvTemplateWithDefault creates an EnvTemplate with a default value
func NewEnvTemplateWithDefault(name string, defaultValue string) *EnvTemplate {
	return &EnvTemplate{
		Name:         name,
		DefaultValue: &defaultValue,
	}
}

// Value returns the value which is retrieved from system or the default value if exist
func (et EnvTemplate) Value() (string, bool) {
	value, ok := os.LookupEnv(et.Name)
	if !ok && et.DefaultValue != nil {
		return *et.DefaultValue, true
	}
	return value, ok
}

// String implements the Stringer interface
func (et EnvTemplate) String() string {
	if et.DefaultValue == nil {
		return fmt.Sprintf("{{%s}}", et.Name)
	}
	return fmt.Sprintf("{{%s:-%s}}", et.Name, *et.DefaultValue)
}

// FindEnvTemplate finds one environment template from string
func FindEnvTemplate(input string) *EnvTemplate {
	matches := envVariableRegex.FindStringSubmatch(input)
	return parseEnvTemplateFromMatches(matches)
}

// FindAllEnvTemplates finds all unique environment templates from string
func FindAllEnvTemplates(input string) []EnvTemplate {
	matches := envVariableRegex.FindAllStringSubmatch(input, -1)
	var results []EnvTemplate
	for _, item := range matches {
		env := parseEnvTemplateFromMatches(item)
		if env == nil {
			continue
		}
		doesExist := false
		for _, result := range results {
			if env.String() == result.String() {
				doesExist = true
				break
			}
		}
		if !doesExist {
			results = append(results, *env)
		}
	}
	return results
}

func parseEnvTemplateFromMatches(matches []string) *EnvTemplate {
	if len(matches) != 4 {
		return nil
	}
	result := &EnvTemplate{
		Name: matches[1],
	}

	if matches[2] != "" {
		result.DefaultValue = &matches[3]
	}
	return result
}

// ReplaceEnvTemplates replaces env templates in the input string with values
func ReplaceEnvTemplates(input string, envTemplates []EnvTemplate) string {
	for _, env := range envTemplates {
		value, _ := env.Value()
		input = strings.ReplaceAll(input, env.String(), value)
	}
	return input
}
