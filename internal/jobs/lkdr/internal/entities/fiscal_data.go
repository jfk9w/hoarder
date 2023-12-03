package entities

type FiscalDataItem struct {
	ReceiptKey string `json:"-" gorm:"primaryKey"`
	DbIdx      int    `json:"dbIdx" gorm:"primaryKey"`

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

func (fd FiscalData) TableName() string {
	return "fiscal_data"
}
