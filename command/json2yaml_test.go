package command

import (
	"fmt"
	"log/slog"
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestJson2Yaml(t *testing.T) {
	tempDir := t.TempDir()
	outputFilePath := fmt.Sprintf("%s/output.yaml", tempDir)
	Json2Yaml(&Json2YamlCommandArguments{
		File:   "https://petstore.swagger.io/v2/swagger.json",
		Output: outputFilePath,
	}, slog.Default())

	outputBytes, err := os.ReadFile(outputFilePath)
	if err != nil {
		t.Errorf("cannot read the output file at %s", outputFilePath)
		t.FailNow()
	}
	var output any
	if err := yaml.Unmarshal(outputBytes, &output); err != nil {
		t.Errorf("cannot decode the output file yaml at %s", outputFilePath)
		t.FailNow()
	}
}
