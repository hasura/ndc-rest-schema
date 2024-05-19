package utils

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	jsonpatch "github.com/evanphx/json-patch"
	"github.com/hasura/ndc-rest-schema/schema"
	"gopkg.in/yaml.v3"
)

// PatchStrategy represents the patch strategy enum
type PatchStrategy string

const (
	// PatchStrategyMerge the merge strategy enum for [RFC 7396] specification
	//
	// [RFC 7396]: https://datatracker.ietf.org/doc/html/rfc7396
	PatchStrategyMerge PatchStrategy = "merge"
	// PatchStrategyJSON6902 the patch strategy enum for [RFC 6902] specification
	//
	// [RFC 6902]: https://datatracker.ietf.org/doc/html/rfc6902
	PatchStrategyJSON6902 PatchStrategy = "json6902"
)

// PatchHookEvent represents the hook event enum of patch file
type PatchHookEvent string

const (
	// PatchBefore should apply the patch before conversion
	PatchBefore PatchHookEvent = "before"
	// PatchBefore should apply the patch after conversion
	PatchAfter PatchHookEvent = "after"
)

// PatchConfig the configuration for JSON patch
type PatchConfig struct {
	Path     string         `json:"path" yaml:"path"`
	Hook     PatchHookEvent `json:"hook" yaml:"hook"`
	Strategy PatchStrategy  `json:"strategy" yaml:"strategy"`
}

// ApplyPatchToRestSchema applies JSON patches to NDC rest schema and validate the output
func ApplyPatchToRestSchema(input *schema.NDCRestSchema, patchFiles []PatchConfig) (*schema.NDCRestSchema, error) {
	if len(patchFiles) == 0 {
		return input, nil
	}

	bs, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	rawResult, err := ApplyPatchFromRawJSON(bs, patchFiles)
	if err != nil {
		return nil, err
	}

	var result schema.NDCRestSchema
	if err := json.Unmarshal(rawResult, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// ApplyPatch applies patches to the raw bytes input
func ApplyPatch(input []byte, patchFiles []PatchConfig) ([]byte, error) {
	jsonInput, err := convertMaybeYAMLToJSONBytes(input)
	if err != nil {
		return nil, err
	}

	return ApplyPatchFromRawJSON(jsonInput, patchFiles)
}

// ApplyPatchFromRawJSON applies patches to the raw JSON bytes input without validation request
func ApplyPatchFromRawJSON(input []byte, patchFiles []PatchConfig) ([]byte, error) {
	for _, patchFile := range patchFiles {
		walkError := WalkFiles(patchFile.Path, func(data []byte) error {
			jsonPatch, err := convertMaybeYAMLToJSONBytes(data)
			if err != nil {
				return fmt.Errorf("%s: %s", patchFile.Path, err)
			}
			strategy := patchFile.Strategy
			if strategy == "" {
				strategy, err = guessPatchStrategy(jsonPatch)
				if err != nil {
					return fmt.Errorf("%s: %s", patchFile.Path, err)
				}
			}
			switch strategy {
			case PatchStrategyJSON6902:
				patch, err := jsonpatch.DecodePatch(jsonPatch)
				if err != nil {
					return fmt.Errorf("failed to decode patch from file %s: %s", patchFile, err)
				}
				input, err = patch.Apply(input)
				if err != nil {
					return fmt.Errorf("failed to apply patch from file %s: %s", patchFile, err)
				}
			case PatchStrategyMerge:
				input, err = jsonpatch.MergePatch(input, jsonPatch)
				if err != nil {
					return fmt.Errorf("failed to merge JSON patch from file %s: %s", patchFile, err)
				}
			default:
				return fmt.Errorf("invalid JSON path strategy: %s", patchFile.Strategy)
			}

			return nil
		})
		if walkError != nil {
			return nil, walkError
		}
	}

	return input, nil
}

func convertMaybeYAMLToJSONBytes(input []byte) ([]byte, error) {
	runes := []byte(strings.TrimSpace(string(input)))
	if len(runes) <= 0 {
		return nil, errors.New("empty data input")
	}

	if (runes[0] == '{' && runes[len(runes)-1] == '}') || (runes[0] == '[' && runes[len(runes)-1] == ']') {
		return []byte(runes), nil
	}

	var anyOutput any
	if err := yaml.Unmarshal(input, &anyOutput); err != nil {
		return nil, fmt.Errorf("input bytes are not in either yaml or json format: %s", err)
	}
	return json.Marshal(anyOutput)
}

func guessPatchStrategy(runes []byte) (PatchStrategy, error) {
	if len(runes) <= 0 {
		return "", errors.New("empty input")
	}

	if runes[0] == '{' && runes[len(runes)-1] == '}' {
		return PatchStrategyMerge, nil
	}
	if runes[0] == '[' && runes[len(runes)-1] == ']' {
		return PatchStrategyJSON6902, nil
	}
	return "", errors.New("unable to detect patch strategy")
}
