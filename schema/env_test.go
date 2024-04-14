package schema

import (
	"encoding/json"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEnvTemplate(t *testing.T) {
	testCases := []struct {
		input     string
		expected  string
		templates []EnvTemplate
	}{
		{},
		{
			input:    "http://localhost:8080",
			expected: "http://localhost:8080",
		},
		{
			input: "{{SERVER_URL}}",
			templates: []EnvTemplate{
				NewEnvTemplate("SERVER_URL"),
			},
			expected: "",
		},
		{
			input: "{{SERVER_URL:-http://localhost:8080}}",
			templates: []EnvTemplate{
				NewEnvTemplateWithDefault("SERVER_URL", "http://localhost:8080"),
			},
			expected: "http://localhost:8080",
		},
		{
			input: "{{SERVER_URL:-}}",
			templates: []EnvTemplate{
				{
					Name:         "SERVER_URL",
					DefaultValue: toPtr(""),
				},
			},
			expected: "",
		},
		{
			input: "{{SERVER_URL:-http://localhost:8080}},{{SERVER_URL:-http://localhost:8080}},{{SERVER_URL}}",
			templates: []EnvTemplate{
				{
					Name:         "SERVER_URL",
					DefaultValue: toPtr("http://localhost:8080"),
				},
				{
					Name: "SERVER_URL",
				},
			},
			expected: "http://localhost:8080,http://localhost:8080,",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			tmpl := FindEnvTemplate(tc.input)
			if len(tc.templates) == 0 {
				if tmpl != nil {
					t.Errorf("failed to find env template, expected nil, got %s", tmpl)
				}
			} else {
				assertDeepEqual(t, tc.templates[0].String(), tmpl.String())
			}

			templates := FindAllEnvTemplates(tc.input)
			assertDeepEqual(t, tc.templates, templates)
			for i, item := range templates {
				assertDeepEqual(t, tc.templates[i].String(), item.String())
			}
			assertDeepEqual(t, tc.expected, ReplaceEnvTemplates(tc.input, templates))
		})
	}
}

func TestEnvInt(t *testing.T) {
	testCases := []struct {
		input    string
		expected EnvInt
	}{
		{
			input:    `400`,
			expected: EnvInt{value: toPtr[int64](400)},
		},
		{
			input:    `"400"`,
			expected: EnvInt{value: toPtr[int64](400)},
		},
		{
			input: `"{{FOO:-401}}"`,
			expected: EnvInt{
				value: toPtr(int64(401)),
				EnvTemplate: EnvTemplate{
					Name:         "FOO",
					DefaultValue: toPtr("401"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			var result EnvInt
			if err := json.Unmarshal([]byte(tc.input), &result); err != nil {
				t.Error(t, err)
				t.FailNow()
			}
			assertDeepEqual(t, tc.expected.EnvTemplate, result.EnvTemplate)
			assertDeepEqual(t, tc.expected.value, result.value)

			if err := yaml.Unmarshal([]byte(tc.input), &result); err != nil {
				t.Error(t, err)
				t.FailNow()
			}
			assertDeepEqual(t, tc.expected.EnvTemplate, result.EnvTemplate)
			assertDeepEqual(t, tc.expected.value, result.value)
		})
	}
}

func TestEnvInts(t *testing.T) {
	testCases := []struct {
		input    string
		expected EnvInts
	}{
		{
			input:    `[400, 401, 403]`,
			expected: EnvInts{value: []int64{400, 401, 403}},
		},
		{
			input:    `"400, 401, 403"`,
			expected: EnvInts{value: []int64{400, 401, 403}},
		},
		{
			input: `"{{FOO:-400, 401, 403}}"`,
			expected: EnvInts{
				value: []int64{400, 401, 403},
				EnvTemplate: EnvTemplate{
					Name:         "FOO",
					DefaultValue: toPtr("400, 401, 403"),
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			var result EnvInts
			if err := json.Unmarshal([]byte(tc.input), &result); err != nil {
				t.Error(t, err)
				t.FailNow()
			}
			assertDeepEqual(t, tc.expected.EnvTemplate, result.EnvTemplate)
			assertDeepEqual(t, tc.expected.value, result.value)

			if err := yaml.Unmarshal([]byte(tc.input), &result); err != nil {
				t.Error(t, err)
				t.FailNow()
			}
			assertDeepEqual(t, tc.expected.EnvTemplate, result.EnvTemplate)
			assertDeepEqual(t, tc.expected.value, result.value)
		})
	}
}
