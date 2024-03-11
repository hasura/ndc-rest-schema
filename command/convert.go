package command

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/hasura/ndc-schema-tool/openapi"
	"github.com/hasura/ndc-schema-tool/types"
	"github.com/hasura/ndc-schema-tool/utils"
)

// ConvertCommandArguments represent available command arguments for the convert command
type ConvertCommandArguments struct {
	File     string `help:"File path needs to be converted." short:"f" required:""`
	Output   string `help:"The location where the ndc schema file will be generated" short:"o" default:"output.json"`
	Spec     string `help:"The API specification of the file, is one of openapi3, openapi2" default:"openapi3"`
	Rest     bool   `help:"Return REST NDC schema extension" default:"false"`
	LogLevel string `help:"Log level." enum:"trace,debug,info,warn,error" default:"info"`
}

// ConvertToNDCSchema converts to NDC REST schema from file
func ConvertToNDCSchema(args *ConvertCommandArguments) {
	logger, err := initLogger(args.LogLevel)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
		return
	}

	logger.Info("converting to NDC schema")
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
}

func initLogger(logLevel string) (*slog.Logger, error) {
	var level slog.Level
	err := level.UnmarshalText([]byte(strings.ToUpper(logLevel)))
	if err != nil {
		return nil, err
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
	}))
	slog.SetDefault(logger)

	return logger, nil
}
