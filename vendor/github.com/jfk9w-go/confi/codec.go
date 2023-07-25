package confi

import (
	"encoding/gob"
	"encoding/json"
	"io"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Codec struct {
	MarshalFn   func(value any, writer io.Writer) error
	UnmarshalFn func(reader io.Reader, value any) error
}

func (c Codec) Marshal(value any, writer io.Writer) error {
	data, err := yaml.Marshal(value)
	if err != nil {
		return errors.Wrap(err, "marshal value to yaml")
	}

	values := make(map[string]any)
	if err := yaml.Unmarshal(data, &values); err != nil {
		return errors.Wrap(err, "unmarshal values from yaml")
	}

	if err := c.MarshalFn(values, writer); err != nil {
		return errors.Wrap(err, "marshal values")
	}

	return nil
}

func (c Codec) Unmarshal(reader io.Reader, value any) error {
	values := make(map[string]any)
	if err := c.UnmarshalFn(reader, &values); err != nil {
		return errors.Wrap(err, "unmarshal values")
	}

	data, err := yaml.Marshal(values)
	if err != nil {
		return errors.Wrap(err, "marshal values to yaml")
	}

	if err := yaml.Unmarshal(data, value); err != nil {
		return errors.Wrap(err, "unmarshal value from yaml")
	}

	return nil
}

var JSON = Codec{
	MarshalFn: func(value any, writer io.Writer) error {
		encoder := json.NewEncoder(writer)
		encoder.SetIndent("", "  ")
		return encoder.Encode(value)
	},

	UnmarshalFn: func(reader io.Reader, value any) error { return json.NewDecoder(reader).Decode(value) },
}

var YAML = Codec{
	MarshalFn:   func(value any, writer io.Writer) error { return yaml.NewEncoder(writer).Encode(value) },
	UnmarshalFn: func(reader io.Reader, value any) error { return yaml.NewDecoder(reader).Decode(value) },
}

var Gob = Codec{
	MarshalFn:   func(value any, writer io.Writer) error { return gob.NewEncoder(writer).Encode(value) },
	UnmarshalFn: func(reader io.Reader, value any) error { return gob.NewDecoder(reader).Decode(value) },
}

var Codecs = map[string]Codec{
	"json": JSON,
	"yaml": YAML,
	"yml":  YAML,
	"gob":  Gob,
}

type Format string

func (f Format) SchemaEnum() any {
	var formats []string
	for format := range Codecs {
		formats = append(formats, format)
	}

	return formats
}
