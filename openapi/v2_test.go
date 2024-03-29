package openapi

import (
	"encoding/json"
	"errors"
	"os"
	"testing"

	"github.com/hasura/ndc-rest-schema/schema"
)

func TestOpenAPIv2ToRESTSchema(t *testing.T) {

	testCases := []struct {
		Name     string
		Source   string
		Options  *ConvertOptions
		Expected string
	}{
		{
			Name:     "jsonplaceholder",
			Source:   "testdata/jsonplaceholder/swagger.json",
			Expected: "testdata/jsonplaceholder/expected.json",
			Options: &ConvertOptions{
				TrimPrefix: "/v1",
			},
		},
		{
			Name:     "petstore2",
			Source:   "testdata/petstore2/swagger.json",
			Expected: "testdata/petstore2/expected.json",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			sourceBytes, err := os.ReadFile(tc.Source)
			assertNoError(t, err)

			expectedBytes, err := os.ReadFile(tc.Expected)
			assertNoError(t, err)
			var expected schema.NDCRestSchema
			assertNoError(t, json.Unmarshal(expectedBytes, &expected))

			output, errs := OpenAPIv2ToNDCSchema(sourceBytes, tc.Options)
			if output == nil {
				t.Error(errors.Join(errs...))
				t.FailNow()
			}

			assertDeepEqual(t, expected.Collections, output.Collections, "Collections")
			assertDeepEqual(t, expected.Settings, output.Settings, "Settings")
			assertDeepEqual(t, expected.ScalarTypes, output.ScalarTypes, "ScalarTypes")
			assertDeepEqual(t, expected.ObjectTypes, output.ObjectTypes, "ObjectTypes")
			assertDeepEqual(t, expected.Procedures, output.Procedures, "Procedures")
			assertDeepEqual(t, expected.Functions, output.Functions, "Functions")
		})
	}

	t.Run("failure_empty", func(t *testing.T) {
		_, err := OpenAPIv2ToNDCSchema([]byte(""), nil)
		assertError(t, errors.Join(err...), "there is nothing in the spec, it's empty")
	})
}
