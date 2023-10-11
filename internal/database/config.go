package database

import (
	"sort"

	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var drivers = map[Driver]func(string) gorm.Dialector{
	"postgres": postgres.Open,
	"mysql":    mysql.Open,
	"sqlite":   sqlite.Open,
}

type Driver string

func (Driver) SchemaEnum() any {
	var names []string
	for name := range drivers {
		names = append(names, string(name))
	}

	sort.Strings(names)
	return names
}

type Config struct {
	Driver Driver `yaml:"driver"`
	DSN    string `yaml:"dsn" examples:"\"file::memory:?cache=shared\", \"host=localhost port=5432 user=postgres password=postgres dbname=postgres search_path=public\""`
}
