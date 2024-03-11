package main

import (
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/alecthomas/kong"
	"github.com/hasura/ndc-schema-tool/command"
	"github.com/hasura/ndc-schema-tool/version"
)

var cli struct {
	LogLevel  string                            `help:"Log level." enum:"debug,info,warn,error" default:"info"`
	Convert   command.ConvertCommandArguments   `cmd:"" help:"Convert API spec to NDC schema. For example:\n ndc-schema-tool convert -f petstore.yaml -o petstore.json"`
	Json2Yaml command.Json2YamlCommandArguments `cmd:"" name:"json2yaml" help:"Convert JSON file to YAML. For example:\n ndc-schema-tool json2yaml -f petstore.json -o petstore.yaml"`
	Version   struct{}                          `cmd:"" help:"Print the CLI version."`
}

func main() {
	cmd := kong.Parse(&cli, kong.UsageOnError())
	logger, err := initLogger(cli.LogLevel)
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
		return
	}

	switch cmd.Command() {
	case "convert":
		command.ConvertToNDCSchema(&cli.Convert, logger)
	case "json2yaml":
		command.Json2Yaml(&cli.Json2Yaml, logger)
	case "version":
		_, _ = fmt.Print(version.BuildVersion)
	default:
		logger.Error(fmt.Sprintf("unknown command <%s>", cmd.Command()))
		os.Exit(1)
	}
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
