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
		// go run . convert -f ./openapi/testdata/jsonplaceholder/swagger.json -o ./openapi/testdata/jsonplaceholder/expected.json --spec openapi2 --trim-prefix /v1
		{
			Name:     "jsonplaceholder",
			Source:   "testdata/jsonplaceholder/swagger.json",
			Expected: "testdata/jsonplaceholder/expected.json",
			Options: &ConvertOptions{
				TrimPrefix: "/v1",
			},
		},
		// go run . convert -f ./openapi/testdata/petstore2/swagger.json -o ./openapi/testdata/petstore2/expected.json --spec openapi2
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

			assertRESTSchemaEqual(t, &expected, output)
		})
	}

	t.Run("failure_empty", func(t *testing.T) {
		_, err := OpenAPIv2ToNDCSchema([]byte(""), nil)
		assertError(t, errors.Join(err...), "there is nothing in the spec, it's empty")
	})
}
