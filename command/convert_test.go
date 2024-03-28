package command

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/hasura/ndc-rest-schema/schema"
)

func TestConvertToNDCSchema(t *testing.T) {
	testCases := []struct {
		name     string
		filePath string
		spec     schema.SchemaSpecType
		pure     bool
		noOutput bool
		format   schema.SchemaFileFormat
		errorMsg string
	}{
		{
			name:     "file_not_found",
			filePath: "foo.json",
			spec:     schema.OpenAPIv3Spec,
			errorMsg: "failed to read content from foo.json: open foo.json: no such file or directory",
		},
		{
			name:     "invalid_spec",
			filePath: "../openapi/testdata/petstore3/source.json",
			spec:     schema.SchemaSpecType("unknown"),
			errorMsg: "invalid spec unknown, expected",
		},
		{
			name:     "openapi3",
			filePath: "../openapi/testdata/petstore3/source.json",
			spec:     schema.OpenAPIv3Spec,
		},
		{
			name:     "openapi2",
			filePath: "../openapi/testdata/petstore2/swagger.json",
			spec:     schema.OpenAPIv2Spec,
			pure:     true,
			noOutput: true,
			format:   schema.SchemaFileYAML,
		},
		{
			name:     "invalid_output_format",
			filePath: "../openapi/testdata/petstore2/swagger.json",
			spec:     schema.OpenAPIv2Spec,
			pure:     true,
			noOutput: true,
			errorMsg: "invalid SchemaFileFormat",
		},
		{
			name:     "openapi3_failure",
			filePath: "../openapi/testdata/petstore2/swagger.json",
			spec:     schema.OpenAPIv3Spec,
			errorMsg: "unable to build openapi document, supplied spec is a different version (oas2)",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var outputFilePath string
			if !tc.noOutput {
				tempDir := t.TempDir()
				outputFilePath = fmt.Sprintf("%s/output.json", tempDir)
			}
			err := ConvertToNDCSchema(&ConvertCommandArguments{
				File:   tc.filePath,
				Output: outputFilePath,
				Pure:   tc.pure,
				Spec:   string(tc.spec),
				Format: string(tc.format),
			}, slog.Default())

			if tc.errorMsg != "" {
				assertError(t, err, tc.errorMsg)
				return
			}

			assertNoError(t, err)
			if tc.noOutput {
				return
			}
			outputBytes, err := os.ReadFile(outputFilePath)
			if err != nil {
				t.Errorf("cannot read the output file at %s", outputFilePath)
				t.FailNow()
			}
			var output schema.NDCRestSchema
			if err := json.Unmarshal(outputBytes, &output); err != nil {
				t.Errorf("cannot decode the output file json at %s", outputFilePath)
				t.FailNow()
			}
		})
	}
}

func assertNoError(t *testing.T, err error) {
	if err != nil {
		t.Errorf("expected no error, got: %s", err)
		panic(err)
	}
}

func assertError(t *testing.T, err error, message string) {
	if err == nil {
		t.Error("expected error, got nil")
		t.FailNow()
	} else if !strings.Contains(err.Error(), message) {
		t.Errorf("expected error with content: %s, got: %s", err.Error(), message)
		t.FailNow()
	}
}
