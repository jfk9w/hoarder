package lkdr

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"time"

	"github.com/pkg/errors"

	"github.com/jfk9w-go/based"
)

var dateTimeLocation = &based.Lazy[*time.Location]{
	Fn: func(ctx context.Context) (*time.Location, error) {
		return time.LoadLocation("Europe/Moscow")
	},
}

type DateTime time.Time

const dateTimeLayout = "2006-01-02T15:04:05"

func (dt DateTime) Time() time.Time {
	return time.Time(dt)
}

func (dt DateTime) MarshalJSON() ([]byte, error) {
	location, err := dateTimeLocation.Get(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "load location")
	}

	str := time.Time(dt).In(location).Format(dateTimeLayout)
	return json.Marshal(str)
}

func (dt *DateTime) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	location, err := dateTimeLocation.Get(context.Background())
	if err != nil {
		return errors.Wrap(err, "load location")
	}

	value, err := time.ParseInLocation(dateTimeLayout, str, location)
	if err != nil {
		return err
	}

	*dt = DateTime(value)
	return nil
}

func (dt DateTime) GormDataType() string {
	return "time"
}

func (dt DateTime) Value() (driver.Value, error) {
	return time.Time(dt), nil
}

func (dt *DateTime) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		*dt = DateTime(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type DateTimeTZ time.Time

func (dt DateTimeTZ) Time() time.Time {
	return time.Time(dt)
}

const dateTimeTZLayout = "2006-01-02T15:04:05.999Z"

func (dt DateTimeTZ) MarshalJSON() ([]byte, error) {
	str := dt.Time().Format(dateTimeTZLayout)
	return json.Marshal(str)
}

func (dt *DateTimeTZ) UnmarshalJSON(data []byte) error {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}

	value, err := time.Parse(dateTimeTZLayout, str)
	if err != nil {
		return err
	}

	*dt = DateTimeTZ(value)
	return nil
}

func (dt DateTimeTZ) GormDataType() string {
	return "time"
}

func (dt DateTimeTZ) Value() (driver.Value, error) {
	return time.Time(dt), nil
}

func (dt *DateTimeTZ) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		*dt = DateTimeTZ(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type User struct {
	Phone string `gorm:"primaryKey"`
	Name  string `gorm:"index"`
}

type Tokens struct {
	UserPhone string `json:"-" gorm:"primaryKey"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	RefreshToken          string      `json:"refreshToken"`
	RefreshTokenExpiresIn *DateTimeTZ `json:"refreshTokenExpiresIn,omitempty"`
	Token                 string      `json:"token"`
	TokenExpireIn         DateTimeTZ  `json:"tokenExpireIn"`
}

type Brand struct {
	Description string  `json:"description"`
	Id          int64   `json:"id" gorm:"primaryKey"`
	Image       *string `json:"image"`
	Name        string  `json:"name"`
}

type Receipt struct {
	UserPhone string `json:"-" gorm:"index"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	BrandId *int64 `json:"brandId" gorm:"index"`
	Brand   *Brand `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	Buyer                string   `json:"buyer"`
	BuyerType            string   `json:"buyerType"`
	CreatedDate          DateTime `json:"createdDate" gorm:"index"`
	FiscalDocumentNumber string   `json:"fiscalDocumentNumber"`
	FiscalDriveNumber    string   `json:"fiscalDriveNumber"`
	Key                  string   `json:"key" gorm:"primaryKey"`
	KktOwner             string   `json:"kktOwner"`
	KktOwnerInn          string   `json:"kktOwnerInn"`
	ReceiveDate          DateTime `json:"receiveDate" gorm:"index"`
	TotalSum             string   `json:"totalSum"`
}

type FiscalDataItem struct {
	ReceiptKey string `json:"-" gorm:"primaryKey"`
	Position   int    `json:"-" gorm:"primaryKey"`

	Name        string  `json:"name"`
	Nds         int     `json:"nds"`
	PaymentType int     `json:"paymentType"`
	Price       float64 `json:"price"`
	ProductType int     `json:"productType"`
	ProviderInn *string `json:"providerInn"`
	Quantity    float64 `json:"quantity"`
	Sum         float64 `json:"sum"`
}

type FiscalData struct {
	ReceiptKey string  `json:"-" gorm:"primaryKey"`
	Receipt    Receipt `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	BuyerAddress            string           `json:"buyerAddress"`
	CashTotalSum            float64          `json:"cashTotalSum"`
	CreditSum               float64          `json:"creditSum"`
	DateTime                DateTime         `json:"dateTime"`
	EcashTotalSum           float64          `json:"ecashTotalSum"`
	FiscalDocumentFormatVer string           `json:"fiscalDocumentFormatVer"`
	FiscalDocumentNumber    int64            `json:"fiscalDocumentNumber"`
	FiscalDriveNumber       string           `json:"fiscalDriveNumber"`
	FiscalSign              string           `json:"fiscalSign"`
	InternetSign            *int             `json:"internetSign"`
	Items                   []FiscalDataItem `json:"items" gorm:"constraint:OnDelete:CASCADE;foreignKey:ReceiptKey"`
	KktRegId                string           `json:"kktRegId"`
	MachineNumber           *string          `json:"machineNumber"`
	Nds10                   *float64         `json:"nds10"`
	Nds18                   *float64         `json:"nds18"`
	OperationType           int              `json:"operationType"`
	Operator                *string          `json:"operator"`
	PrepaidSum              float64          `json:"prepaidSum"`
	ProvisionSum            float64          `json:"provisionSum"`
	RequestNumber           int64            `json:"requestNumber"`
	RetailPlace             string           `json:"retailPlace"`
	RetailPlaceAddress      *string          `json:"retailPlaceAddress"`
	ShiftNumber             int64            `json:"shiftNumber"`
	TaxationType            int              `json:"taxationType"`
	TotalSum                float64          `json:"totalSum"`
	User                    *string          `json:"user"`
	UserInn                 string           `json:"userInn"`
}
