package command

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"testing"

	"github.com/hasura/ndc-schema-tool/types"
)

func TestConvertToNDCSchema(t *testing.T) {
	tempDir := t.TempDir()
	outputFilePath := fmt.Sprintf("%s/output.json", tempDir)
	ConvertToNDCSchema(&ConvertCommandArguments{
		File:   "https://raw.githubusercontent.com/stripe/openapi/master/openapi/spec3.yaml",
		Output: outputFilePath,
		Rest:   true,
		Spec:   string(types.OpenAPIv3Spec),
	}, slog.Default())

	outputBytes, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Errorf("cannot read the output file at %s", outputFilePath)
		t.FailNow()
	}
	var output types.NDCRestSchema
	if err := json.Unmarshal(outputBytes, &output); err != nil {
		t.Errorf("cannot decode the output file json at %s", outputFilePath)
		t.FailNow()
	}
}
