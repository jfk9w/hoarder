package database

import (
	"github.com/pkg/errors"
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
	var names []Driver
	for name := range drivers {
		names = append(names, name)
	}

	return names
}

type Config struct {
	Driver Driver `yaml:"driver"`
	DSN    string `yaml:"dsn" examples:"\"file::memory:?cache=shared\""`
}

func Open(cfg Config) (*gorm.DB, error) {
	driver, ok := drivers[cfg.Driver]
	if !ok {
		return nil, errors.Errorf("unsupported driver: %s", cfg.Driver)
	}

	return gorm.Open(driver(cfg.DSN))
}
