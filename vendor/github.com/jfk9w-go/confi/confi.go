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

func SpecifyType(value any) any {
	switch typed := value.(type) {
	case string:
		if v, err := strconv.ParseBool(typed); err == nil {
			return v
		} else if v, err := strconv.ParseUint(typed, 10, 64); err == nil {
			return v
		} else if v, err := strconv.ParseInt(typed, 10, 64); err == nil {
			return v
		} else if v, err := strconv.ParseFloat(typed, 64); err == nil {
			return v
		}

		return value

	case []any:
		values := make([]any, len(typed))
		for i, value := range typed {
			values[i] = SpecifyType(value)
		}

		return values

	case map[string]any:
		values := make(map[any]any, len(typed))
		for key, value := range typed {
			values[SpecifyType(key)] = SpecifyType(value)
		}

		return values
	}

	return value
}
