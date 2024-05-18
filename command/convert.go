package command

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/hasura/ndc-rest-schema/openapi"
	"github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
)

// ConvertCommandArguments represent available command arguments for the convert command
type ConvertCommandArguments struct {
	File        string            `help:"File path needs to be converted." short:"f" required:""`
	Output      string            `help:"The location where the ndc schema file will be generated. Print to stdout if not set" short:"o"`
	Spec        string            `help:"The API specification of the file, is one of openapi3, openapi2" default:"openapi3"`
	Format      string            `help:"The output format, is one of json, yaml. If the output is set, automatically detect the format in the output file extension" default:"json"`
	Pure        bool              `help:"Return the pure NDC schema only" default:"false"`
	TrimPrefix  string            `help:"Trim the prefix in URL, e.g. /v1"`
	EnvPrefix   string            `help:"The environment variable prefix for security values, e.g. PET_STORE"`
	MethodAlias map[string]string `help:"Alias names for HTTP method. Used for prefix renaming, e.g. getUsers, postUser"`
	PatchBefore []string          `help:"Patch files to be applied into the input file before converting"`
	PatchAfter  []string          `help:"Patch files to be applied into the input file after converting"`
}

// ConvertToNDCSchema converts to NDC REST schema from file
func ConvertToNDCSchema(args *ConvertCommandArguments, logger *slog.Logger) error {
	logger.Debug(
		"converting OpenAPI definition to NDC REST schema",
		slog.String("file", args.File),
		slog.String("output", args.Output),
		slog.String("spec", args.Spec),
		slog.String("format", args.Format),
		slog.String("trim_prefix", args.TrimPrefix),
		slog.String("env_prefix", args.EnvPrefix),
		slog.Any("patch_before", args.PatchBefore),
		slog.Any("patch_after", args.PatchAfter),
		slog.Bool("pure", args.Pure),
	)
	rawContent, err := utils.ReadFileFromPath(args.File)
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	rawContent, err = utils.ApplyPatch(rawContent, stringSliceToPatchConfigs(args.PatchBefore))
	if err != nil {
		logger.Error(err.Error())
		return err
	}

	var result *schema.NDCRestSchema
	var errs []error
	options := &openapi.ConvertOptions{
		MethodAlias: args.MethodAlias,
		TrimPrefix:  args.TrimPrefix,
		EnvPrefix:   args.EnvPrefix,
		Logger:      logger,
	}
	switch args.Spec {
	case string(schema.OpenAPIv3Spec), string(schema.OAS3Spec):
		result, errs = openapi.OpenAPIv3ToNDCSchema(rawContent, options)
	case string(schema.OpenAPIv2Spec), string(schema.OAS2Spec):
		result, errs = openapi.OpenAPIv2ToNDCSchema(rawContent, options)
	default:
		err := fmt.Errorf("invalid spec %s, expected %+v", args.Spec, []schema.SchemaSpecType{schema.OpenAPIv3Spec, schema.OpenAPIv2Spec})
		logger.Error(err.Error())
		return err
	}
	if len(errs) > 0 {
		logger.Error(errors.Join(errs...).Error())
	}
	if result == nil {
		return errors.Join(errs...)
	}

	result, err = utils.ApplyPatchToRestSchema(result, stringSliceToPatchConfigs(args.PatchAfter))
	if args.Output != "" {
		if args.Pure {
			err = utils.WriteSchemaFile(args.Output, result.ToSchemaResponse())
		} else {
			err = utils.WriteSchemaFile(args.Output, result)
		}
		if err != nil {
			logger.Error("failed to write schema file", slog.String("error", err.Error()))
			return err
		}

		logger.Info("generated successfully")
		return nil
	}

	// print to stderr
	format, err := schema.ParseSchemaFileFormat(args.Format)
	if err != nil {
		logger.Error("failed to parse format", slog.Any("error", err))
		return err
	}

	var rawResult any = result
	if args.Pure {
		rawResult = result.ToSchemaResponse()
	}

	resultBytes, err := utils.MarshalSchema(rawResult, format)
	if err != nil {
		logger.Error("failed to encode schema", slog.Any("error", err))
		return err
	}

	fmt.Print(string(resultBytes))
	return nil
}

func stringSliceToPatchConfigs(input []string) []utils.PatchConfig {
	result := make([]utils.PatchConfig, len(input))
	for i, str := range input {
		result[i] = utils.PatchConfig{Path: str}
	}
	return result
}
