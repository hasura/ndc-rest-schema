package openapi

import (
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/hasura/ndc-rest-schema/schema"
)

func TestOpenAPIv3ToRESTSchema(t *testing.T) {

	testCases := []struct {
		Name     string
		Source   string
		Expected string
	}{
		{
			Name:     "petstore3",
			Source:   "testdata/petstore3/source.json",
			Expected: "testdata/petstore3/expected.json",
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

			output, errs := OpenAPIv3ToNDCSchema(sourceBytes, &ConvertOptions{
				EnvPrefix: "PET_STORE",
			})
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
		_, err := OpenAPIv3ToNDCSchema([]byte(""), nil)
		assertError(t, errors.Join(err...), "there is nothing in the spec, it's empty")
	})
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
		t.Errorf("%s: not equal.\nexpected: %s\ngot			: %s", strings.Join(msgs, " "), string(expectedJson), string(realityJson))
		t.FailNow()
	}
}
