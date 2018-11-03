package transform

import (
	"bytes"
	"errors"
	"strings"

	"github.com/gohugoio/hugo/parser"
	"github.com/gohugoio/hugo/parser/metadecoders"
	"github.com/spf13/cast"
)

// Remarshal is used in the Hugo documentation to convert configuration
// examples from YAML to JSON, TOML (and possibly the other way around).
// The is primarily a helper for the Hugo docs site.
// It is not a general purpose YAML to TOML converter etc., and may
// change without notice if it serves a purpose in the docs.
// Format is one of json, yaml or toml.
func (ns *Namespace) Remarshal(format string, data interface{}) (string, error) {
	from, err := cast.ToStringE(data)
	if err != nil {
		return "", err
	}

	from = strings.TrimSpace(from)
	format = strings.TrimSpace(strings.ToLower(format))

	if from == "" {
		return "", nil
	}

	mark, err := toFormatMark(format)
	if err != nil {
		return "", err
	}

	fromFormat, err := detectFormat(from)
	if err != nil {
		return "", err
	}

	meta, err := metadecoders.UnmarshalToMap([]byte(from), fromFormat)

	var result bytes.Buffer
	if err := parser.InterfaceToConfig(meta, mark, &result); err != nil {
		return "", err
	}

	return result.String(), nil
}

func toFormatMark(format string) (metadecoders.Format, error) {
	if f := metadecoders.FormatFromString(format); f != "" {
		return f, nil
	}

	return "", errors.New("failed to detect target data serialization format")
}

func detectFormat(data string) (metadecoders.Format, error) {
	jsonIdx := strings.Index(data, "{")
	yamlIdx := strings.Index(data, ":")
	tomlIdx := strings.Index(data, "=")

	if jsonIdx != -1 && (yamlIdx == -1 || jsonIdx < yamlIdx) && (tomlIdx == -1 || jsonIdx < tomlIdx) {
		return metadecoders.JSON, nil
	}

	if yamlIdx != -1 && (tomlIdx == -1 || yamlIdx < tomlIdx) {
		return metadecoders.YAML, nil
	}

	if tomlIdx != -1 {
		return metadecoders.TOML, nil
	}

	return "", errors.New("failed to detect data serialization format")

}
