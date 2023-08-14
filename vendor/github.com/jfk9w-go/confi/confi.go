package confi

import (
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

func Get[T any](ctx context.Context, appName string) (*T, *Schema, error) {
	replacer := strings.NewReplacer(`-`, `_`, `.`, `_`)
	provider := &DefaultSourceProvider{
		EnvPrefix: replacer.Replace(appName) + "_",
		Env:       os.Environ(),
		Args:      os.Args[1:],
		Stdin:     os.Stdin,
	}

	return FromProvider[T](ctx, provider)
}

func FromProvider[T any](ctx context.Context, provider SourceProvider) (*T, *Schema, error) {
	sources, err := provider.GetSources(ctx)
	if err != nil {
		return nil, nil, errors.Wrap(err, "get sources")
	}

	var config T

	schema, err := GenerateSchema(config)
	if err != nil {
		return nil, nil, errors.Wrapf(err, "generate schema")
	}

	for _, source := range sources {
		values, err := source.GetValues(ctx)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "get values from %s", source)
		}

		data, err := yaml.Marshal(SpecifyType(values))
		if err != nil {
			return nil, nil, errors.Wrapf(err, "marshal values from %s to yaml", source)
		}

		if err := yaml.Unmarshal(data, &config); err != nil {
			return nil, nil, errors.Wrapf(err, "unmarshal values from %s from yaml", source)
		}
	}

	if err := schema.ApplyDefaults(&config); err != nil {
		return nil, nil, errors.Wrap(err, "apply defaults")
	}

	return &config, schema, nil
}

func testFloat(source string, target float64) bool {
	tokens := strings.Split(source, ".")
	var prec int
	if len(tokens) > 1 {
		prec = len(tokens[1])
	}

	return strconv.FormatFloat(target, 'f', prec, 64) == source
}

func SpecifyType(source any) any {
	switch source := source.(type) {
	case string:
		if target, err := strconv.ParseBool(source); err == nil && strconv.FormatBool(target) == source {
			return true
		} else if target, err := strconv.ParseUint(source, 10, 64); err == nil && strconv.FormatUint(target, 10) == source {
			return target
		} else if target, err := strconv.ParseInt(source, 10, 64); err == nil && strconv.FormatInt(target, 10) == source {
			return target
		} else if target, err := strconv.ParseFloat(source, 64); err == nil && testFloat(source, target) {
			return target
		}

		return source

	case []any:
		target := make([]any, len(source))
		for i, value := range source {
			target[i] = SpecifyType(value)
		}

		return target

	case map[string]any:
		target := make(map[any]any, len(source))
		for key, value := range source {
			target[SpecifyType(key)] = SpecifyType(value)
		}

		return target
	}

	return source
}
