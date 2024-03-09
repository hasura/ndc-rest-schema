package openapi

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var (
	bracketRegexp       = regexp.MustCompile(`[\{\}]`)
	toUnderscoreRegexp  = regexp.MustCompile(`[^\w]`)
	schemaRefNameRegexp = regexp.MustCompile(`#/components/schemas/([\w]+)`)

	errParameterSchemaEmpty = errors.New("parameter schema is empty")
)

const (
	ContentTypeHeader = "Content-Type"
	ContentTypeJSON   = "application/json"
)

func buildPathMethodName(apiPath string, method string) string {
	encodedPath := strings.ToLower(toUnderscoreRegexp.ReplaceAllString(bracketRegexp.ReplaceAllString(strings.TrimLeft(apiPath, "/"), ""), "_"))
	if method == "get" {
		return encodedPath
	}
	return fmt.Sprintf("%s_%s", method, encodedPath)
}

func getSchemaRefTypeName(name string) string {
	result := schemaRefNameRegexp.FindStringSubmatch(name)
	if len(result) < 2 {
		return ""
	}
	return result[1]
}
