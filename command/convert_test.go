package command

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hasura/ndc-rest-schema/schema"
)

var nopLogger = slog.New(slog.NewTextHandler(io.Discard, &slog.HandlerOptions{}))

func TestConvertToNDCSchema(t *testing.T) {
	testCases := []struct {
		name        string
		filePath    string
		spec        schema.SchemaSpecType
		pure        bool
		noOutput    bool
		format      schema.SchemaFileFormat
		patchBefore []string
		patchAfter  []string
		expected    string
		errorMsg    string
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
		{
			name:        "patch",
			filePath:    "../openapi/testdata/onesignal/source.json",
			spec:        schema.OpenAPIv3Spec,
			patchBefore: []string{"../openapi/testdata/onesignal/patch-before.json"},
			patchAfter:  []string{"../openapi/testdata/onesignal/patch-after.json"},
			expected:    "../openapi/testdata/onesignal/expected-patch.json",
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
				File:        tc.filePath,
				Output:      outputFilePath,
				Pure:        tc.pure,
				Spec:        string(tc.spec),
				Format:      string(tc.format),
				PatchBefore: tc.patchBefore,
				PatchAfter:  tc.patchAfter,
			}, nopLogger)

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
			if tc.expected == "" {
				return
			}

			expectedBytes, err := os.ReadFile(tc.expected)
			if err != nil {
				t.Errorf("cannot read the expected file at %s", outputFilePath)
				t.FailNow()
			}
			var expectedSchema schema.NDCRestSchema
			if err := json.Unmarshal(expectedBytes, &expectedSchema); err != nil {
				t.Errorf("cannot decode the output file json at %s", tc.expected)
				t.FailNow()
			}
			assertDeepEqual(t, expectedSchema.Settings, output.Settings)
			assertDeepEqual(t, expectedSchema.Functions, output.Functions)
			assertDeepEqual(t, expectedSchema.Procedures, output.Procedures)
			assertDeepEqual(t, expectedSchema.ScalarTypes, output.ScalarTypes)
			assertDeepEqual(t, expectedSchema.ObjectTypes, output.ObjectTypes)
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

func assertDeepEqual(t *testing.T, expected any, reality any, msgs ...string) {
	if reflect.DeepEqual(expected, reality) {
		return
	}

	expectedJson, _ := json.Marshal(expected)
	realityJson, _ := json.Marshal(reality)

	var expected1, reality1 any
	assertNoError(t, json.Unmarshal(expectedJson, &expected1))
	assertNoError(t, json.Unmarshal(realityJson, &reality1))

	if !reflect.DeepEqual(expected1, reality1) {
		t.Errorf("%s: not equal.\nexpected: %s\ngot     : %s", strings.Join(msgs, " "), string(expectedJson), string(realityJson))
		t.FailNow()
	}
}
