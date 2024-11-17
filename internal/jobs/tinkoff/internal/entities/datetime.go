package entities

import (
	"database/sql/driver"
	"time"

	"github.com/jfk9w-go/tinkoff-api/v2"
	"github.com/pkg/errors"
)

type Milliseconds struct {
	tinkoff.Milliseconds
}

func (ms Milliseconds) GormDataType() string {
	return "time"
}

func (ms Milliseconds) Value() (driver.Value, error) {
	return ms.Time(), nil
}

func (ms *Milliseconds) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		ms.Milliseconds = tinkoff.Milliseconds(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type ReceiptDateTime struct {
	tinkoff.ReceiptDateTime
}

func (dt ReceiptDateTime) GormDataType() string {
	return "time"
}

func (dt ReceiptDateTime) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *ReceiptDateTime) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.ReceiptDateTime = tinkoff.ReceiptDateTime(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type DateTimeMilliOffset struct {
	tinkoff.DateTimeMilliOffset
}

func (dt DateTimeMilliOffset) GormDataType() string {
	return "time"
}

func (dt DateTimeMilliOffset) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *DateTimeMilliOffset) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.DateTimeMilliOffset = tinkoff.DateTimeMilliOffset(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type DateTime struct {
	tinkoff.DateTime
}

func (dt DateTime) GormDataType() string {
	return "time"
}

func (dt DateTime) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *DateTime) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.DateTime = tinkoff.DateTime(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type Date struct {
	tinkoff.Date
}

func (d Date) GormDataType() string {
	return "date"
}

func (d Date) Value() (driver.Value, error) {
	return d.Time(), nil
}

func (d *Date) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		d.Date = tinkoff.Date(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type InvestCandleDate struct {
	tinkoff.InvestCandleDate
}

func (d InvestCandleDate) GormDataType() string {
	return "time"
}

func (d InvestCandleDate) Value() (driver.Value, error) {
	return d.Time(), nil
}

func (d *InvestCandleDate) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		d.InvestCandleDate = tinkoff.InvestCandleDate(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}
