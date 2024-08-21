package common

import (
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type UserMap[K, V comparable] struct {
	direct  map[K][]V
	reverse map[V]K
}

func (m *UserMap[K, V]) UnmarshalYAML(node *yaml.Node) error {
	if err := node.Decode(&m.direct); err != nil {
		return err
	}

	m.reverse = make(map[V]K)
	for k, vs := range m.direct {
		for _, v := range vs {
			if e, ok := m.reverse[v]; ok {
				return errors.Errorf("duplicate entries for key [%v]: [%v] and [%v]", v, e, k)
			}

			m.reverse[v] = k
		}
	}

	return nil
}

func (m *UserMap[K, V]) Reverse() map[V]K {
	return m.reverse
}
