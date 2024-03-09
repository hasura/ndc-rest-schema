package utils

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/hasura/ndc-schema-tool/types"
	"gopkg.in/yaml.v3"
)

// WriteSchemaFile writes the NDC REST schema to file
func WriteSchemaFile(outputPath string, content any) error {

	format, err := types.ParseSchemaFileFormat(strings.TrimLeft(filepath.Ext(outputPath), "."))
	if err != nil {
		return err
	}
	var fileBuffer bytes.Buffer
	switch format {
	case types.SchemaFileJSON:
		encoder := json.NewEncoder(&fileBuffer)
		encoder.SetIndent("", "  ")

		if err := encoder.Encode(content); err != nil {
			return fmt.Errorf("failed to encode NDC REST schema: %s", err)
		}
	case types.SchemaFileYAML:
		encoder := yaml.NewEncoder(&fileBuffer)
		encoder.SetIndent(2)
		if err := encoder.Encode(content); err != nil {
			return fmt.Errorf("failed to encode NDC REST schema: %s", err)
		}
	default:
		return errors.New("invalid schema file format. Accept json or yaml")
	}

	return os.WriteFile(outputPath, fileBuffer.Bytes(), 0664)
}

// ReadFileFromPath read file content from either file path or URL
func ReadFileFromPath(filePath string) ([]byte, error) {
	var result []byte

	fileURL, err := url.Parse(filePath)
	if err == nil && slices.Contains([]string{"http", "https"}, strings.ToLower(fileURL.Scheme)) {
		resp, err := http.DefaultClient.Get(filePath)
		if err != nil {
			return nil, err
		}

		if resp.Body != nil {
			result, err = io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("failed to read content from %s: %s", filePath, err)
			}
		}
		if resp.StatusCode != http.StatusOK {
			errorMsg := string(result)
			if errorMsg == "" {
				errorMsg = resp.Status
			}
			return nil, fmt.Errorf("failed to download file from %s: %s", filePath, errorMsg)
		}
	} else {
		result, err = os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to read content from %s: %s", filePath, err)
		}
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("failed to read file from %s: no content", filePath)
	}

	return result, nil
}
