package entities

import "encoding/json"

type Category struct {
	Id   string `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"index"`
}

type Location struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	DbIdx       int    `json:"dbIdx" gorm:"primaryKey"`

	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type LoyaltyAmount struct {
	Value               float64 `json:"value"`
	LoyaltyProgramId    string  `json:"loyaltyProgramId"`
	Loyalty             string  `json:"loyalty"`
	Name                string  `json:"name"`
	LoyaltySteps        uint8   `json:"loyaltySteps"`
	LoyaltyPointsId     uint8   `json:"loyaltyPointsId"`
	LoyaltyPointsName   string  `json:"loyaltyPointsName"`
	LoyaltyImagine      bool    `json:"loyaltyImagine"`
	PartialCompensation bool    `json:"partialCompensation"`
}

type LoyaltyBonus struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	DbIdx       int    `json:"dbIdx" gorm:"primaryKey"`

	Description      string        `json:"description"`
	Icon             string        `json:"icon"`
	LoyaltyType      string        `json:"loyaltyType"`
	Amount           LoyaltyAmount `json:"amount" gorm:"embedded"`
	CompensationType string        `json:"compensationType"`
}

type Region struct {
	Country    *string `json:"country,omitempty"`
	City       *string `json:"city,omitempty"`
	Address    *string `json:"address,omitempty"`
	Zip        *string `json:"zip,omitempty"`
	AddressRus *string `json:"addressRus,omitempty"`
}

type Merchant struct {
	Name   string  `json:"name" gorm:"index"`
	Region *Region `json:"region,omitempty" gorm:"embedded"`
}

type SpendingCategory struct {
	Id   string `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"index"`

	FireflyId *string `json:"-" gorm:"<-:false;index"`
}

func (sc SpendingCategory) TableName() string {
	return "spending_categories"
}

type Brand struct {
	Name          string  `json:"name" gorm:"index"`
	BaseTextColor *string `json:"baseTextColor,omitempty"`
	Logo          *string `json:"logo,omitempty"`
	Id            string  `json:"id" gorm:"primaryKey"`
	RoundedLogo   bool    `json:"roundedLogo"`
	BaseColor     *string `json:"baseColor,omitempty"`
	LogoFile      *string `json:"logoFile,omitempty"`
	Link          *string `json:"link,omitempty"`
	SvgLogoFile   *string `json:"svgLogoFile"`
}

type AdditionalInfo struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	DbIdx       int    `json:"dbIdx" gorm:"primaryKey"`

	FieldName  string `json:"fieldName"`
	FieldValue string `json:"fieldValue"`
}

type LoyaltyPaymentAmount struct {
	LoyaltyAmount `gorm:"embedded"`
	Price         float64 `json:"price"`
}

type LoyaltyPayment struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	DbIdx       int    `json:"dbIdx" gorm:"primaryKey"`

	Amount   LoyaltyPaymentAmount `json:"amount" gorm:"embedded"`
	Status   string               `json:"status"`
	SoldTime *Milliseconds        `json:"soldTime"`
}

type LoyaltyBonusSummary struct {
	Amount float64 `json:"amount"`
}

type PaymentFeeAmount struct {
	FeeAmountCurrencyCode uint     `json:"-" gorm:"index"`
	FeeAmountCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	FeeAmountValue float64 `json:"value"`
}

type PaymentFieldValue struct {
	PaymentId string `gorm:"primaryKey"`
	Key       string `gorm:"primaryKey"`
	Value     string
}

type PaymentFieldsValues []PaymentFieldValue

func (fvs PaymentFieldsValues) MarshalJSON() ([]byte, error) {
	values := make(map[string]string)
	for _, fv := range fvs {
		values[fv.Key] = fv.Value
	}

	return json.Marshal(values)
}

func (fvs *PaymentFieldsValues) UnmarshalJSON(data []byte) error {
	var values map[string]string
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}

	for key, value := range values {
		*fvs = append(*fvs, PaymentFieldValue{
			Key:   key,
			Value: value,
		})
	}

	return nil
}

type Payment struct {
	SourceIsQr      bool                `json:"sourceIsQr"`
	BankAccountId   string              `json:"bankAccountId" gorm:"index"`
	PaymentId       string              `json:"paymentId" gorm:"primaryKey"`
	ProviderGroupId *string             `json:"providerGroupId,omitempty"`
	PaymentType     string              `json:"paymentType"`
	FeeAmount       *PaymentFeeAmount   `json:"feeAmount,omitempty" gorm:"embedded"`
	ProviderId      string              `json:"providerId"`
	HasPaymentOrder bool                `json:"hasPaymentOrder"`
	Comment         *string             `json:"comment,omitempty"`
	IsQrPayment     bool                `json:"isQrPayment"`
	FieldsValues    PaymentFieldsValues `json:"fieldsValues"`
	//Repeatable         bool    `json:"repeatable"`	// отличается у входящих и исходящих платежей
	CardNumber         string  `json:"cardNumber"`
	TemplateId         *string `json:"templateId,omitempty"`
	TemplateIsFavorite *bool   `json:"templateIsFavorite,omitempty"`
}

type Subgroup struct {
	Id   string  `json:"id" gorm:"primaryKey"`
	Name *string `json:"name,omitempty" gorm:"index"`
}

type OperationCashbackAmount struct {
	CashbackCurrencyCode uint     `json:"-" gorm:"index"`
	CashbackCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	CashbackValue float64 `json:"value"`
}

type OperationAmount struct {
	CurrencyCode uint     `json:"-" gorm:"index"`
	Currency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	Value float64 `json:"value"`
}

type OperationAccountAmount struct {
	AccountCurrencyCode uint     `json:"-" gorm:"index"`
	AccountCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	AccountValue float64 `json:"value"`
}

type Operation struct {
	AccountId string  `json:"account" gorm:"index"`
	Account   Account `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	BrandId *string `json:"-" gorm:"index"`
	Brand   *Brand  `json:"brand,omitempty" gorm:"constraint:OnDelete:CASCADE"`

	SpendingCategoryId string           `json:"-" gorm:"index"`
	SpendingCategory   SpendingCategory `json:"spendingCategory" gorm:"constraint:OnDelete:CASCADE"`

	CategoryId string   `json:"-" gorm:"index"`
	Category   Category `json:"category" gorm:"constraint:OnDelete:CASCADE"`

	SubgroupId *string   `json:"-" gorm:"index"`
	Subgroup   *Subgroup `json:"subgroup,omitempty" gorm:"constraint:OnDelete:CASCADE"`

	PaymentId *string  `json:"-" gorm:"index"`
	Payment   *Payment `json:"payment,omitempty" gorm:"constraint:OnDelete:CASCADE"`

	IsDispute              bool                    `json:"isDispute"`
	IsOffline              bool                    `json:"isOffline"`
	HasStatement           bool                    `json:"hasStatement"`
	IsSuspicious           bool                    `json:"isSuspicious"`
	AuthorizationId        *string                 `json:"authorizationId,omitempty"`
	IsInner                bool                    `json:"isInner" gorm:"index"`
	Id                     string                  `json:"id" gorm:"primaryKey"`
	Status                 string                  `json:"status" gorm:"index"`
	OperationTransferred   bool                    `json:"operationTransferred"`
	IdSourceType           string                  `json:"idSourceType"`
	HasShoppingReceipt     *bool                   `json:"hasShoppingReceipt,omitempty" gorm:"index"`
	Type                   string                  `json:"type" gorm:"index"`
	Locations              []Location              `json:"locations,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	LoyaltyBonus           []LoyaltyBonus          `json:"loyaltyBonus,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	CashbackAmount         OperationCashbackAmount `json:"cashbackAmount" gorm:"embedded"`
	AuthMessage            *string                 `json:"authMessage,omitempty"`
	Description            string                  `json:"description"`
	IsTemplatable          bool                    `json:"isTemplatable"`
	Cashback               float64                 `json:"cashback"`
	Amount                 OperationAmount         `json:"amount" gorm:"embedded"`
	OperationTime          Milliseconds            `json:"operationTime" gorm:"index"`
	IsHce                  bool                    `json:"isHce"`
	Mcc                    uint                    `json:"mcc"`
	AdditionalInfo         []AdditionalInfo        `json:"additionalInfo,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	VirtualPaymentType     uint8                   `json:"virtualPaymentType"`
	Ucid                   *string                 `json:"ucid,omitempty"`
	Merchant               *Merchant               `json:"merchant,omitempty" gorm:"embedded;embeddedPrefix:merchant_"`
	Card                   *string                 `json:"card,omitempty" gorm:"index"`
	LoyaltyPayment         []LoyaltyPayment        `json:"loyaltyPayment,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	TrancheCreationAllowed bool                    `json:"trancheCreationAllowed"`
	Group                  *string                 `json:"group,omitempty"`
	MccString              string                  `json:"mccString"`
	CardPresent            bool                    `json:"cardPresent"`
	IsExternalCard         bool                    `json:"isExternalCard"`
	CardNumber             *string                 `json:"cardNumber,omitempty"`
	AccountAmount          OperationAccountAmount  `json:"accountAmount" gorm:"embedded"`
	LoyaltyBonusSummary    *LoyaltyBonusSummary    `json:"loyaltyBonusSummary,omitempty" gorm:"embedded;embeddedPrefix:loyalty_bonus_summary_"`
	TypeSerno              *uint                   `json:"typeSerno"`
	OperationPaymentType   *string                 `json:"operationPaymentType,omitempty"`
	DebitingTime           *Milliseconds           `json:"debitingTime,omitempty" gorm:"index"`
	PosId                  *string                 `json:"posId,omitempty"`
	Subcategory            *string                 `json:"subcategory,omitempty" gorm:"index"`
	SenderAgreement        *string                 `json:"senderAgreement,omitempty"`
	PointOfSaleId          *uint64                 `json:"pointOfSaleId,omitempty"`
	Compensation           *string                 `json:"compensation,omitempty"`
	InstallmentStatus      *string                 `json:"installmentStatus,omitempty"`
	SenderDetails          *string                 `json:"senderDetails,omitempty"`
	PartnerType            *string                 `json:"partnerType,omitempty"`
	Nomination             *string                 `json:"nomination,omitempty"`
	Message                *string                 `json:"message,omitempty"`
	TrancheId              *string                 `json:"trancheId,omitempty"`

	FireflyId *string `json:"-" gorm:"<-:false;index"`
}

func (o Operation) TableName() string {
	return "operations"
}
