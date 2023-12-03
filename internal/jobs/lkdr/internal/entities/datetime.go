package entities

import (
	"database/sql/driver"
	"time"

	"github.com/jfk9w-go/lkdr-api"
	"github.com/pkg/errors"
)

type DateTime struct {
	lkdr.DateTime
}

func (dt DateTime) GormDataType() string {
	return "time"
}

func (dt DateTime) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *DateTime) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.DateTime = lkdr.DateTime(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type DateTimeTZ struct {
	lkdr.DateTimeTZ
}

func (dt DateTimeTZ) GormDataType() string {
	return "time"
}

func (dt DateTimeTZ) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *DateTimeTZ) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.DateTimeTZ = lkdr.DateTimeTZ(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}
