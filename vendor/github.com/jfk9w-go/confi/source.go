package confi

import (
	"context"

	"github.com/pkg/errors"
)

type Source interface {
	GetValues(ctx context.Context) (map[string]any, error)
}

type PropertySource []Property

func (s PropertySource) GetValues(ctx context.Context) (map[string]any, error) {
	values := make(map[string]any)
	for _, prop := range s {
		last := len(prop.Path) - 1
		container := values
		for i, el := range prop.Path {
			if i == last {
				if existing, ok := container[el].(map[string]any); existing == nil || !ok {
					container[el] = prop.Value
				}

				continue
			}

			value := container[el]
			if _, ok := value.(map[string]any); !ok {
				container[el] = make(map[string]any)
			}

			container, _ = container[el].(map[string]any)
		}
	}

	return values, nil
}

type InputSource struct {
	Input  Input
	Format string
}

func (s InputSource) GetValues(ctx context.Context) (map[string]any, error) {
	reader, err := s.Input.Reader()
	if err != nil {
		return nil, errors.Wrap(err, "open input")
	}

	defer CloseQuietly(reader)

	if s.Format == "properties" {
		props, err := readProperties(reader)
		if err != nil {
			return nil, errors.Wrap(err, "read properties")
		}

		return PropertySource(props).GetValues(ctx)
	}

	codec, ok := Codecs[s.Format]
	if !ok {
		return nil, errors.Wrapf(err, "no codec found for %s", s.Format)
	}

	values := make(map[string]any)
	if err := codec.UnmarshalFn(reader, &values); err != nil {
		return nil, err
	}

	return values, nil
}
