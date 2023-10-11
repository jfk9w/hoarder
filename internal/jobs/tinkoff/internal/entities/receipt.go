package entities

type ReceiptItem struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	DbIdx       int    `json:"dbIdx" gorm:"primaryKey"`

	Name     string   `json:"name" gorm:"index"`
	Price    float64  `json:"price"`
	Sum      float64  `json:"sum"`
	Quantity float64  `json:"quantity"`
	NdsRate  *uint8   `json:"ndsRate"`
	Nds      *uint8   `json:"nds"`
	Nds10    *float64 `json:"nds10,omitempty"`
	Nds18    *float64 `json:"nds18,omitempty"`
	BrandId  *uint64  `json:"brand_id,omitempty"`
	GoodId   *uint64  `json:"good_id,omitempty"`
}

type Receipt struct {
	OperationId string    `json:"-" gorm:"primaryKey"`
	Operation   Operation `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	RetailPlace             *string       `json:"retailPlace,omitempty"`
	RetailPlaceAddress      *string       `json:"retailPlaceAddress,omitempty"`
	CreditSum               *float64      `json:"creditSum,omitempty"`
	ProvisionSum            *float64      `json:"provisionSum,omitempty"`
	FiscalDriveNumber       *uint64       `json:"fiscalDriveNumber,omitempty"`
	OperationType           uint8         `json:"operationType"`
	CashTotalSum            float64       `json:"cashTotalSum"`
	ShiftNumber             uint          `json:"shiftNumber"`
	KktRegId                string        `json:"kktRegId"`
	Items                   []ReceiptItem `json:"items" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	TotalSum                float64       `json:"totalSum"`
	EcashTotalSum           float64       `json:"ecashTotalSum"`
	Nds10                   *float64      `json:"nds10,omitempty"`
	Nds18                   *float64      `json:"nds18,omitempty"`
	UserInn                 string        `json:"userInn"`
	DateTime                Seconds       `json:"dateTime"`
	TaxationType            uint8         `json:"taxationType"`
	PrepaidSum              *float64      `json:"prepaidSum,omitempty"`
	FiscalSign              uint64        `json:"fiscalSign"`
	RequestNumber           uint          `json:"requestNumber"`
	Operator                *string       `json:"operator,omitempty"`
	AppliedTaxationType     uint8         `json:"appliedTaxationType"`
	FiscalDocumentNumber    uint64        `json:"fiscalDocumentNumber"`
	User                    *string       `json:"user,omitempty"`
	FiscalDriveNumberString string        `json:"fiscalDriveNumberString"`
}

func (r Receipt) TableName() string {
	return "receipts"
}
