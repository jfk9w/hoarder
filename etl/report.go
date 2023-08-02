package etl

import (
	"fmt"
	"strings"
)

type StatGroup struct {
}

type Stats struct {
	stats  map[string]*Stats
	keys   []string
	synced int
	warns  []string
	error  string
	leaf   bool
}

func (s *Stats) Get(key string, leaf bool) *Stats {
	if s.stats == nil {
		s.stats = make(map[string]*Stats)
	}

	stat, ok := s.stats[key]
	if !ok {
		stat = &Stats{leaf: leaf}
		s.stats[key] = stat
		s.keys = append(s.keys, key)
	}

	return stat
}

func (s *Stats) Add(count int) {
	s.synced += count
}

func (s *Stats) Warnf(pattern string, args ...any) {
	s.warns = append(s.warns, fmt.Sprintf(pattern, args...))
}

func (s *Stats) Error(err error) {
	if err != nil {
		s.error = err.Error()
	}
}

func (s *Stats) IsError() bool {
	return s.error != ""
}

func (s *Stats) String() string {
	var b strings.Builder
	if s.leaf {
		for _, key := range s.keys {
			stats := s.stats[key]
			b.WriteString(fmt.Sprintf("%s • %d done", key, stats.synced))
		}
	}

	return b.String()
}
