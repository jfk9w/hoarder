package entities

import (
	"database/sql/driver"
	"time"

	tbank "github.com/jfk9w-go/tbank-api"
	"github.com/pkg/errors"
)

type Milliseconds struct {
	tbank.Milliseconds
}

func (ms Milliseconds) GormDataType() string {
	return "time"
}

func (ms Milliseconds) Value() (driver.Value, error) {
	return ms.Time(), nil
}

func (ms *Milliseconds) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		ms.Milliseconds = tbank.Milliseconds(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type ReceiptDateTime struct {
	tbank.ReceiptDateTime
}

func (dt ReceiptDateTime) GormDataType() string {
	return "time"
}

func (dt ReceiptDateTime) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *ReceiptDateTime) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.ReceiptDateTime = tbank.ReceiptDateTime(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type DateTimeMilliOffset struct {
	tbank.DateTimeMilliOffset
}

func (dt DateTimeMilliOffset) GormDataType() string {
	return "time"
}

func (dt DateTimeMilliOffset) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *DateTimeMilliOffset) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.DateTimeMilliOffset = tbank.DateTimeMilliOffset(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type DateTime struct {
	tbank.DateTime
}

func (dt DateTime) GormDataType() string {
	return "time"
}

func (dt DateTime) Value() (driver.Value, error) {
	return dt.Time(), nil
}

func (dt *DateTime) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		dt.DateTime = tbank.DateTime(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type Date struct {
	tbank.Date
}

func (d Date) GormDataType() string {
	return "date"
}

func (d Date) Value() (driver.Value, error) {
	return d.Time(), nil
}

func (d *Date) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		d.Date = tbank.Date(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type InvestCandleDate struct {
	tbank.InvestCandleDate
}

func (d InvestCandleDate) GormDataType() string {
	return "time"
}

func (d InvestCandleDate) Value() (driver.Value, error) {
	return d.Time(), nil
}

func (d *InvestCandleDate) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		d.InvestCandleDate = tbank.InvestCandleDate(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}
