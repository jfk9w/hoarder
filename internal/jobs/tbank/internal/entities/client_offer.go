package entities

type ClientOfferEssenceMccCode struct {
	ClientOfferEssenceId string `json:"-" gorm:"primaryKey"`
	MccCode              string `json:"-" gorm:"primaryKey"`
}

type ClientOfferEssence struct {
	ClientOfferId string `json:"-" gorm:"index"`

	SpendingCategoryId *string `json:"-" gorm:"index"`
	BrandId            *string `json:"-" gorm:"index"`

	MccCodes []ClientOfferEssenceMccCode `json:"-" gorm:"constraint:OnDelete:CASCADE;foreignKey:ClientOfferEssenceId"`

	Name         string `json:"name"`
	Description  string `json:"description"`
	BusinessType uint   `json:"businessType"`
	IsActive     bool   `json:"isActive"`
	BaseColor    string `json:"baseColor"`
	//MccCodes     []string `json:"mccCodes,omitempty"`
	Logo         string  `json:"logo"`
	ExternalCode string  `json:"externalCode"`
	ExternalId   string  `json:"externalId"`
	Id           string  `json:"id" gorm:"primaryKey"`
	Percent      float64 `json:"percent"`
}

type ClientOfferAttributes struct {
	NotificationFlag bool `json:"notificationFlag"`
}

type ClientOfferAccount struct {
	ClientOfferId string `json:"-" gorm:"primaryKey"`
	AccountId     string `json:"-" gorm:"primaryKey"`
}

type ClientOffer struct {
	UserPhone string `json:"-" gorm:"index"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	Accounts []ClientOfferAccount `json:"-" gorm:"constraint:OnDelete:CASCADE;foreignKey:ClientOfferId"`

	TypeCode              string                `json:"typeCode"`
	AvailableEssenceCount uint                  `json:"availableEssenceCount"`
	ActiveTo              Milliseconds          `json:"activeTo"`
	Attributes            ClientOfferAttributes `json:"attributes" gorm:"embedded"`
	ActiveFrom            Milliseconds          `json:"activeFrom"`
	Essences              []ClientOfferEssence  `json:"essences" gorm:"constraint:OnDelete:CASCADE;foreignKey:ClientOfferId"`
	DisplayTo             Milliseconds          `json:"displayTo"`
	DisplayFrom           Milliseconds          `json:"displayFrom"`
	//AccountIds            []string              `json:"accountIds"`
	Id string `json:"id" gorm:"primaryKey"`
}

func (o ClientOffer) TableName() string {
	return "client_offers"
}
