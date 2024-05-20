package command

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hasura/ndc-rest-schema/openapi"
	"github.com/hasura/ndc-rest-schema/schema"
	"github.com/hasura/ndc-rest-schema/utils"
	"gopkg.in/yaml.v3"
)

// ConvertCommandArguments represent available command arguments for the convert command
type ConvertCommandArguments struct {
	File        string            `help:"File path needs to be converted." short:"f"`
	Config      string            `help:"Path of the config file." short:"c"`
	Output      string            `help:"The location where the ndc schema file will be generated. Print to stdout if not set" short:"o"`
	Spec        string            `help:"The API specification of the file, is one of oas3 (openapi3), oas2 (openapi2)" default:"oas3"`
	Format      string            `help:"The output format, is one of json, yaml. If the output is set, automatically detect the format in the output file extension" default:"json"`
	Pure        bool              `help:"Return the pure NDC schema only" default:"false"`
	TrimPrefix  string            `help:"Trim the prefix in URL, e.g. /v1"`
	EnvPrefix   string            `help:"The environment variable prefix for security values, e.g. PET_STORE"`
	MethodAlias map[string]string `help:"Alias names for HTTP method. Used for prefix renaming, e.g. getUsers, postUser"`
	PatchBefore []string          `help:"Patch files to be applied into the input file before converting"`
	PatchAfter  []string          `help:"Patch files to be applied into the input file after converting"`
}

// ConvertToNDCSchema converts to NDC REST schema from file
func CommandConvertToNDCSchema(args *ConvertCommandArguments, logger *slog.Logger) error {
	logger.Debug(
		"converting the document to NDC REST schema",
		slog.String("file", args.File),
		slog.String("config", args.Config),
		slog.String("output", args.Output),
		slog.String("spec", args.Spec),
		slog.String("format", args.Format),
		slog.String("trim_prefix", args.TrimPrefix),
		slog.String("env_prefix", args.EnvPrefix),
		slog.Any("patch_before", args.PatchBefore),
		slog.Any("patch_after", args.PatchAfter),
		slog.Bool("pure", args.Pure),
	)

	if args.File == "" && args.Config == "" {
		err := errors.New("--config or --file argument is required")
		logger.Error(err.Error())
		return err
	}

	var config ConvertConfig
	if args.Config != "" {
		rawConfig, err := utils.ReadFileFromPath(args.Config)
		if err != nil {
			logger.Error(err.Error())
			return err
		}
		if err := yaml.Unmarshal(rawConfig, &config); err != nil {
			logger.Error(err.Error())
			return err
		}
	}

	mergeConvertArgumentsToConfig(&config, args)
	result, err := ConvertToNDCSchema(&config, logger)

	if err != nil {
		logger.Error(err.Error())
		return err
	}

	if config.Output != "" {
		if config.Pure {
			err = utils.WriteSchemaFile(config.Output, result.ToSchemaResponse())
		} else {
			err = utils.WriteSchemaFile(config.Output, result)
		}
		if err != nil {
			logger.Error("failed to write schema file", slog.String("error", err.Error()))
			return err
		}

		logger.Info("generated successfully")
		return nil
	}

	// print to stderr
	format := schema.SchemaFileJSON
	if args.Format != "" {
		format, err = schema.ParseSchemaFileFormat(args.Format)
		if err != nil {
			logger.Error("failed to parse format", slog.Any("error", err))
			return err
		}
	}

	var rawResult any = result
	if config.Pure {
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

// ConvertConfig represents the content of convert config file
type ConvertConfig struct {
	File        string                `json:"file" yaml:"file"`
	Spec        schema.SchemaSpecType `json:"spec" yaml:"spec"`
	MethodAlias map[string]string     `json:"methodAlias" yaml:"methodAlias"`
	TrimPrefix  string                `json:"trimPrefix" yaml:"trimPrefix"`
	EnvPrefix   string                `json:"envPrefix" yaml:"envPrefix"`
	Pure        bool                  `json:"pure" yaml:"pure"`
	PatchBefore []utils.PatchConfig   `json:"patchBefore" yaml:"patchBefore"`
	PatchAfter  []utils.PatchConfig   `json:"patchAfter" yaml:"patchAfter"`
	Output      string                `json:"output" yaml:"output"`
}

// ConvertToNDCSchema converts to NDC REST schema from config
func ConvertToNDCSchema(config *ConvertConfig, logger *slog.Logger) (*schema.NDCRestSchema, error) {

	rawContent, err := utils.ReadFileFromPath(config.File)
	if err != nil {
		return nil, err
	}

	rawContent, err = utils.ApplyPatch(rawContent, config.PatchBefore)
	if err != nil {
		return nil, err
	}

	var result *schema.NDCRestSchema
	var errs []error
	options := openapi.ConvertOptions{
		MethodAlias: config.MethodAlias,
		TrimPrefix:  config.TrimPrefix,
		EnvPrefix:   config.EnvPrefix,
		Logger:      logger,
	}
	switch config.Spec {
	case schema.OpenAPIv3Spec, schema.OAS3Spec:
		result, errs = openapi.OpenAPIv3ToNDCSchema(rawContent, options)
	case schema.OpenAPIv2Spec, (schema.OAS2Spec):
		result, errs = openapi.OpenAPIv2ToNDCSchema(rawContent, options)
	case schema.NDCSpec:
		if err := json.Unmarshal(rawContent, &result); err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("invalid spec %s, expected %+v", config.Spec, []schema.SchemaSpecType{schema.OpenAPIv3Spec, schema.OpenAPIv2Spec})
	}

	if result == nil {
		return nil, errors.Join(errs...)
	} else if len(errs) > 0 {
		logger.Error(errors.Join(errs...).Error())
	}

	return utils.ApplyPatchToRestSchema(result, config.PatchAfter)
}

func mergeConvertArgumentsToConfig(config *ConvertConfig, args *ConvertCommandArguments) {
	if args.File != "" {
		config.File = args.File
	}
	if args.Spec != "" {
		config.Spec = schema.SchemaSpecType(args.Spec)
	}
	if len(args.MethodAlias) > 0 {
		config.MethodAlias = args.MethodAlias
	}
	if args.TrimPrefix != "" {
		config.TrimPrefix = args.TrimPrefix
	}
	if args.EnvPrefix != "" {
		config.EnvPrefix = args.EnvPrefix
	}
	if args.Output != "" {
		config.Output = args.Output
	}

	if args.Pure {
		config.Pure = args.Pure
	}
	if len(args.PatchBefore) > 0 {
		config.PatchBefore = make([]utils.PatchConfig, len(args.PatchBefore))
		for i, p := range args.PatchBefore {
			config.PatchBefore[i] = utils.PatchConfig{
				Path: p,
			}
		}
	}
	if len(args.PatchAfter) > 0 {
		config.PatchAfter = make([]utils.PatchConfig, len(args.PatchAfter))
		for i, p := range args.PatchAfter {
			config.PatchAfter[i] = utils.PatchConfig{
				Path: p,
			}
		}
	}
}
