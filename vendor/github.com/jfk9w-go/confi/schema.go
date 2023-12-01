package confi

import (
	"reflect"
	"strings"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type formatted interface {
	SchemaFormat() string
}

type patterned interface {
	SchemaPattern() string
}

type enum interface {
	SchemaEnum() any
}

type Schema struct {
	Type                 string            `yaml:"type"`
	Items                *Schema           `yaml:"items,omitempty"`
	Properties           map[string]Schema `yaml:"properties,omitempty"`
	AdditionalProperties any               `yaml:"additionalProperties,omitempty"`
	Required             []string          `yaml:"required,omitempty"`

	// properties below are applied to primitive or inner types
	Enum             any     `yaml:"enum,omitempty" prop:"inner,array"`
	Examples         any     `yaml:"examples,omitempty" prop:"inner,array"`
	Pattern          string  `yaml:"pattern,omitempty" prop:"inner"`
	Format           string  `yaml:"format,omitempty" prop:"inner" alias:"fmt"`
	Minimum          any     `yaml:"minimum,omitempty" prop:"inner" alias:"min"`
	ExclusiveMinimum any     `yaml:"exclusiveMinimum,omitempty" prop:"inner" alias:"xmin"`
	Maximum          any     `yaml:"maximum,omitempty" prop:"inner" alias:"max"`
	ExclusiveMaximum any     `yaml:"exclusiveMaximum,omitempty" prop:"inner" alias:"xmax"`
	MultipleOf       any     `yaml:"multipleOf,omitempty" prop:"inner" alias:"mul"`
	MinLength        uint64  `yaml:"minLength,omitempty" prop:"inner" alias:"minlen"`
	MaxLength        *uint64 `yaml:"maxLength,omitempty" prop:"inner" alias:"maxlen"`

	// properties below are applied to primitive or outer types
	Description   string  `yaml:"description,omitempty" prop:"outer" alias:"desc,doc"`
	Default       any     `yaml:"default,omitempty" prop:"outer" alias:"def"`
	MinItems      uint64  `yaml:"minItems,omitempty" prop:"outer" alias:"minsize"`
	MaxItems      *uint64 `yaml:"maxItems,omitempty" prop:"outer" alias:"maxsize"`
	UniqueItems   bool    `yaml:"uniqueItems,omitempty" prop:"outer" alias:"unique"`
	MinProperties uint64  `yaml:"minProperties,omitempty" prop:"outer" alias:"minprops"`
	MaxProperties *uint64 `yaml:"maxProperties,omitempty" prop:"outer" alias:"maxprops"`
}

func (s *Schema) ApplyDefaults(source any) error {
	return s.applyDefaults(reflect.ValueOf(source))
}

var errUnaddressable = errors.New(
	`unable to set default value for unaddressable value (use pointer values if this is in a map or remove "default" tag)`)

func (s *Schema) applyDefaults(value reflect.Value) error {
	if s.Default != nil && value.IsZero() {
		if !value.CanAddr() {
			return errUnaddressable
		}

		value.Set(reflect.ValueOf(s.Default))
		return nil
	}

	if value.Kind() == reflect.Ptr {
		if value.IsNil() {
			if s.Default == nil {
				return nil
			}

			if !value.CanAddr() {
				return errUnaddressable
			}

			resolvedType := indirectType(value.Type())
			value.Set(reflect.New(resolvedType))
		}

		value = value.Elem()
	}

	if schema, ok := s.AdditionalProperties.(*Schema); ok {
		for _, key := range value.MapKeys() {
			if err := schema.applyDefaults(value.MapIndex(key)); err != nil {
				return errors.Wrapf(err, "on key %v", key.Interface())
			}
		}

		return nil
	}

	if schema := s.Items; schema != nil {
		for i := 0; i < value.Len(); i++ {
			if err := schema.applyDefaults(value.Index(i)); err != nil {
				return errors.Wrapf(err, "on index %d", i)
			}
		}

		return nil
	}

	if properties := s.Properties; properties != nil {
		for fieldNum := 0; fieldNum < value.NumField(); fieldNum++ {
			field := value.Type().Field(fieldNum)
			options := getYAMLOptions(field)
			if options.inline {
				if err := s.applyDefaults(value.Field(fieldNum)); err != nil {
					return errors.Wrapf(err, "on embedded field %s", field.Name)
				}

				continue
			}

			property := properties[options.name]
			if err := property.applyDefaults(value.Field(fieldNum)); err != nil {
				return errors.Wrapf(err, "on field %s", options.name)
			}
		}
	}

	return nil
}

func GenerateSchema(value any) (*Schema, error) {
	return makeSchema(reflect.TypeOf(value), "")
}

func makeSchema(valueType reflect.Type, tag reflect.StructTag) (*Schema, error) {
	resolvedType := indirectType(valueType)
	value := reflect.New(resolvedType).Elem()
	sourceValue := value
	if valueType.Kind() == reflect.Ptr {
		sourceValue = sourceValue.Addr()
	}

	var node yaml.Node
	if err := node.Encode(sourceValue.Interface()); err != nil {
		return nil, errors.Wrap(err, "encode")
	}

	var elemType reflect.Type
	var s Schema
	switch {
	case node.Tag == "!!str":
		s.Type = "string"
		switch value.Interface().(type) {
		case time.Duration:
			s.Pattern = `(\d+h)?(\d+m)?(\d+s)?(\d+ms)?(\d+µs)?(\d+ns)?`

		default:
			if formatted, ok := sourceValue.Interface().(formatted); ok {
				s.Format = formatted.SchemaFormat()
			}

			if patterned, ok := sourceValue.Interface().(patterned); ok {
				s.Pattern = patterned.SchemaPattern()
			}

			if enum, ok := sourceValue.Interface().(enum); ok {
				s.Enum = enum.SchemaEnum()
			}
		}

	case node.Tag == "!!timestamp":
		s.Type = "string"
		s.Format = "date-time"

	case node.Tag == "!!int":
		switch {
		case value.CanInt(), value.CanUint():
			s.Type = "integer"
		default:
			s.Type = "number"
		}

	case node.Tag == "!!float":
		s.Type = "number"

	case node.Tag == "!!bool":
		s.Type = "boolean"

	case node.Tag == "!!seq" && (resolvedType.Kind() == reflect.Slice || resolvedType.Kind() == reflect.Array):
		elemType = valueType.Elem()
		items, err := makeSchema(elemType, "")
		if err != nil {
			return nil, errors.Wrap(err, "generate items")
		}

		s.Type = "array"
		s.Items = items
		if resolvedType.Kind() == reflect.Array {
			s.MinItems = uint64(value.Len())
			s.MaxItems = pointer.To(uint64(value.Len()))
		}

	case node.Tag == "!!map" && resolvedType.Kind() == reflect.Map:
		elemType = resolvedType.Elem()
		additionalProperties, err := makeSchema(elemType, "")
		if err != nil {
			return nil, errors.Wrap(err, "generate additionalProperties")
		}

		s.Type = "object"
		s.AdditionalProperties = additionalProperties

	case node.Tag == "!!map" && resolvedType.Kind() == reflect.Struct:
		properties, required, err := makeStructSchema(valueType)
		if err != nil {
			return nil, errors.Wrap(err, "generate properties & required")
		}

		s.Type = "object"
		s.Properties = properties
		s.AdditionalProperties = false
		s.Required = required
	}

	if s.Type == "" {
		return nil, errors.Errorf("unable to detect type for %s %s", node.Tag, resolvedType)
	}

	if err := applySchemaProps(&s, tag, valueType, elemType); err != nil {
		return nil, errors.Wrap(err, "apply props")
	}

	return &s, nil
}

func applySchemaProps(s *Schema, tag reflect.StructTag, valueType, elemType reflect.Type) error {
	schema := reflect.ValueOf(s).Elem()
	schemaType := schema.Type()
	for fieldNum := 0; fieldNum < schemaType.NumField(); fieldNum++ {
		var (
			field    = schemaType.Field(fieldNum)
			propType string
			isArray  bool
		)

		for i, option := range strings.Split(field.Tag.Get("prop"), ",") {
			switch {
			case i == 0:
				propType = option
			case option == "array":
				isArray = true
			}
		}

		if propType == "" {
			continue
		}

		aliases := []string{getYAMLOptions(field).name}
		aliases = append(aliases, strings.Split(field.Tag.Get("alias"), ",")...)

		var prop string
		for _, alias := range aliases {
			if tag, ok := tag.Lookup(alias); ok {
				prop = tag
				break
			}
		}

		if prop == "" {
			continue
		}

		targetField := schema.Field(fieldNum)
		var fieldValue reflect.Value
		if field.Type.String() == "interface {}" {
			targetType := valueType
			if propType == "inner" && elemType != nil {
				targetType = elemType
				if s.Type == "array" {
					targetField = reflect.Indirect(reflect.ValueOf(s.Items)).Field(fieldNum)
				} else if s.Type == "object" && s.AdditionalProperties != false {
					targetField = reflect.Indirect(reflect.ValueOf(s.AdditionalProperties)).Field(fieldNum)
				}
			}

			if isArray {
				prop = "[" + prop + "]"
				fieldValue = reflect.New(reflect.SliceOf(targetType))
			} else {
				fieldValue = reflect.New(targetType)
			}
		} else {
			fieldValue = reflect.New(field.Type)
		}

		if err := yaml.Unmarshal([]byte(prop), fieldValue.Interface()); err != nil {
			return errors.Wrapf(err, "unmarshal %s", prop)
		}

		fieldValue = reflect.Indirect(fieldValue)
		targetField.Set(fieldValue)
	}

	return nil
}

func makeStructSchema(valueType reflect.Type) (map[string]Schema, []string, error) {
	var (
		resolvedType = indirectType(valueType)
		properties   = make(map[string]Schema)
		required     []string
	)

	for fieldNum := 0; fieldNum < resolvedType.NumField(); fieldNum++ {
		field := resolvedType.Field(fieldNum)
		if !field.IsExported() {
			continue
		}

		options := getYAMLOptions(field)
		if options.inline {
			embedded, err := makeSchema(field.Type, "")
			if err != nil {
				return nil, nil, errors.Wrapf(err, "generate embedded schema for %s", options.name)
			}

			required = append(required, embedded.Required...)
			for name, property := range embedded.Properties {
				properties[name] = property
			}

			continue
		}

		if !options.omitempty {
			required = append(required, options.name)
		}

		property, err := makeSchema(resolvedType.Field(fieldNum).Type, field.Tag)
		if err != nil {
			return nil, nil, errors.Wrapf(err, "generate schema for %s", options.name)
		}

		properties[options.name] = *property
	}

	return properties, required, nil
}

type yamlOptions struct {
	name      string
	omitempty bool
	inline    bool
}

func getYAMLOptions(field reflect.StructField) (options yamlOptions) {
	for i, option := range strings.Split(field.Tag.Get("yaml"), ",") {
		switch {
		case i == 0:
			if len(option) > 0 {
				options.name = option
			} else {
				options.name = field.Name
			}

		case option == "omitempty":
			options.omitempty = true

		case option == "inline":
			options.inline = true
		}
	}

	return
}

func indirectType(typ reflect.Type) reflect.Type {
	if typ.Kind() == reflect.Ptr {
		typ = typ.Elem()
	}

	return typ
}
