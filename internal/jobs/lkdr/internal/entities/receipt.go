package entities

type Brand struct {
	Description string  `json:"description"`
	Id          int64   `json:"id" gorm:"primaryKey;autoIncrement:false"`
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

func (r Receipt) TableName() string {
	return "receipts"
}
