package common

import (
	"github.com/pkg/errors"
)

type UserMap[K, V comparable] map[K][]V

func (m UserMap[K, V]) Reverse() (map[V]K, error) {
	reverse := make(map[V]K)
	for k, vs := range m {
		for _, v := range vs {
			if e, ok := reverse[v]; ok {
				return nil, errors.Errorf("duplicate entries for key [%v]: [%v] and [%v]", v, e, k)
			}

			reverse[v] = k
		}
	}

	return reverse, nil
}
