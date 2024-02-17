package common

import (
	"errors"
	"sync"
)

type MultiMutex[K comparable] struct {
	keyed  map[K]*sync.Mutex
	global sync.RWMutex
}

func (m *MultiMutex[K]) TryLock(key K) (func(), error) {
	mu := m.get(key)
	if ok := mu.TryLock(); !ok {
		return nil, errors.New("locked")
	}

	return mu.Unlock, nil
}

func (m *MultiMutex[K]) get(key K) *sync.Mutex {
	var (
		mu *sync.Mutex
		ok bool
	)

	m.global.RLock()
	if m.keyed != nil {
		mu, ok = m.keyed[key]
	}

	m.global.RUnlock()
	if ok {
		return mu
	}

	m.global.Lock()
	defer m.global.Unlock()
	if m.keyed == nil {
		m.keyed = make(map[K]*sync.Mutex)
	} else if mu, ok = m.keyed[key]; ok {
		return mu
	}

	mu = new(sync.Mutex)
	m.keyed[key] = mu
	return mu
}
