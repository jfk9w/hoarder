package confi

import (
	"bufio"
	"io"
	"strings"

	"github.com/pkg/errors"
)

type Property struct {
	Path  []string
	Value string
}

func (p Property) Key() string {
	return strings.Join(p.Path, ".")
}

func getProperty(text, sep string, expandBools bool) (*Property, error) {
	tokens := strings.SplitN(text, "=", 2)
	if len(tokens[0]) == 0 {
		return nil, errors.New("empty property name")
	}

	if len(tokens) == 1 {
		if expandBools {
			tokens = append(tokens, "true")
		} else {
			return nil, errors.Errorf("empty property value")
		}
	}

	return &Property{
		Path:  strings.Split(tokens[0], sep),
		Value: tokens[1],
	}, nil
}

func readProperties(reader io.Reader) ([]Property, error) {
	var props []Property
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		line := strings.Trim(scanner.Text(), " ")
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		prop, err := getProperty(line, ".", false)
		if err != nil {
			return nil, errors.Wrapf(err, `"%s""`, line)
		}

		props = append(props, *prop)
	}

	return props, nil
}
