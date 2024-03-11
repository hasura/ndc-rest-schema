package command

import (
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/hasura/ndc-schema-tool/openapi"
	"github.com/hasura/ndc-schema-tool/types"
	"github.com/hasura/ndc-schema-tool/utils"
)

// ConvertCommandArguments represent available command arguments for the convert command
type ConvertCommandArguments struct {
	File   string `help:"File path needs to be converted." short:"f" required:""`
	Output string `help:"The location where the ndc schema file will be generated. Print to stdout if not set" short:"o"`
	Spec   string `help:"The API specification of the file, is one of openapi3, openapi2" default:"openapi3"`
	Format string `help:"The output format, is one of json, yaml. If the output is set, automatically detect the format in the output file extension" default:"json"`
	Rest   bool   `help:"Return REST NDC schema extension" default:"false"`
}

// ConvertToNDCSchema converts to NDC REST schema from file
func ConvertToNDCSchema(args *ConvertCommandArguments, logger *slog.Logger) {
	rawContent, err := utils.ReadFileFromPath(args.File)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
		return
	}

	var result *types.NDCRestSchema
	var errs []error
	switch args.Spec {
	case string(types.OpenAPIv3Spec):
		result, errs = openapi.OpenAPIv3ToNDCSchema(rawContent)
	case string(types.OpenAPIv2Spec):
		result, errs = openapi.OpenAPIv2ToNDCSchema(rawContent)
	default:
		slog.Error(fmt.Sprintf("invalid spec %s, expected %+v", args.Spec, []types.SchemaSpecType{types.OpenAPIv3Spec, types.OpenAPIv2Spec}))
	}
	if len(errs) > 0 {
		logger.Error(errors.Join(errs...).Error())
	}
	if result == nil {
		os.Exit(1)
		return
	}

	if args.Output != "" {
		if args.Rest {
			err = utils.WriteSchemaFile(args.Output, result)
		} else {
			err = utils.WriteSchemaFile(args.Output, result.ToSchemaResponse())
		}
		if err != nil {
			slog.Error("failed to write schema file: %s", err)
			os.Exit(1)
			return
		}

		logger.Info("generated successfully")
		return
	}

	// print to stderr
	format, err := types.ParseSchemaFileFormat(args.Format)
	if err != nil {
		slog.Error("failed to parse format: %s", err)
		os.Exit(1)
		return
	}

	var rawResult any = result
	if !args.Rest {
		rawResult = result.ToSchemaResponse()
	}

	resultBytes, err := utils.MarshalSchema(rawResult, format)
	if err != nil {
		slog.Error("failed to encode schema: %s", err)
		os.Exit(1)
		return
	}

	fmt.Print(string(resultBytes))
}
