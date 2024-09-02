package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/hasura/ndc-rest-schema/command"
	"github.com/hasura/ndc-rest-schema/schema"
	"github.com/invopop/jsonschema"
)

func main() {
	if err := jsonSchemaConvertConfig(); err != nil {
		panic(fmt.Errorf("failed to write jsonschema for ConvertConfig: %s", err))
	}
	if err := jsonSchemaNdcRESTSchema(); err != nil {
		panic(fmt.Errorf("failed to write jsonschema for NDCRestSchema: %s", err))
	}
}

func jsonSchemaConvertConfig() error {
	reflectSchema := jsonschema.Reflect(&command.ConvertConfig{})
	schemaBytes, err := json.MarshalIndent(reflectSchema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("convert-config.jsonschema", schemaBytes, 0644)
}

func jsonSchemaNdcRESTSchema() error {
	reflectSchema := jsonschema.Reflect(&schema.NDCRestSchema{})
	schemaBytes, err := json.MarshalIndent(reflectSchema, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile("ndc-rest-schema.jsonschema", schemaBytes, 0644)
}