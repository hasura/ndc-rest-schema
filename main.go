package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/alecthomas/kong"
	"github.com/hasura/ndc-schema-tool/command"
	"github.com/hasura/ndc-schema-tool/version"
)

var cli struct {
	Convert command.ConvertCommandArguments `cmd:"" help:"Convert API spec to NDC schema. For example:\n ndc-schema-tool convert -f petstore.yaml -o petstore.json"`

	Version struct{} `cmd:"" help:"Print the CLI version."`
}

func main() {
	cmd := kong.Parse(&cli, kong.UsageOnError())
	switch cmd.Command() {
	case "convert":
		command.ConvertToNDCSchema(&cli.Convert)
	case "version":
		_, _ = fmt.Print(version.BuildVersion)
	default:
		slog.Error(fmt.Sprintf("unknown command <%s>", cmd.Command()))
		os.Exit(1)
	}
}
