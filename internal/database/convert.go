package database

import (
	"encoding/json"
	"strings"
)

func ToViaJSON[T any](source any) (target T, err error) {
	data, err := json.Marshal(source)
	if err != nil {
		return
	}

	var value any
	if err = json.Unmarshal(data, &value); err != nil {
		return
	}

	value = trim(value)
	index(value)

	data, err = json.Marshal(value)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &target)
	return
}

func trim(source any) any {
	switch source := source.(type) {
	case map[string]any:
		target := make(map[string]any, len(source))
		for key, value := range source {
			value := trim(value)
			if value == nil {
				continue
			}

			target[key] = value
		}

		return target

	case []any:
		target := make([]any, len(source))
		for i, value := range source {
			target[i] = trim(value)
		}

		return target

	case string:
		target := strings.Trim(source, " ")
		if target == "" {
			return nil
		}

		return target

	default:
		return source
	}
}

func index(source any) {
	switch source := source.(type) {
	case []any:
		for i, value := range source {
			index(value)
			if values, ok := value.(map[string]any); ok {
				values["dbIdx"] = i + 1
			}
		}

	case map[string]any:
		for _, value := range source {
			index(value)
		}
	}
}
