package tinkoff

import (
	"database/sql/driver"
	"time"

	"github.com/jfk9w-go/tinkoff-api"
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

type Seconds struct {
	tinkoff.Seconds
}

func (s Seconds) GormDataType() string {
	return "time"
}

func (s Seconds) Value() (driver.Value, error) {
	return s.Time(), nil
}

func (s *Seconds) Scan(value any) error {
	if value, ok := value.(time.Time); ok {
		s.Seconds = tinkoff.Seconds(value)
		return nil
	}

	return errors.Errorf("expected time.Time, got %T", value)
}

type User struct {
	Phone string `gorm:"primaryKey"`
	Name  string `gorm:"index"`
}

type Session struct {
	UserPhone string `json:"-" gorm:"primaryKey"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	ID string
}

type Currency struct {
	Code    uint   `json:"code" gorm:"primaryKey"`
	Name    string `json:"name" gorm:"index"`
	StrCode string `json:"strCode"`
}

type MoneyAmount struct {
	CurrencyCode string   `json:"-" gorm:"index"`
	Currency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	Value float64 `json:"value"`
}

type MultiCardCluster struct {
	Id string `json:"id"`
}

type Card struct {
	AccountId string `json:"-" gorm:"index"`

	Id               string            `json:"id" gorm:"primaryKey"`
	StatusCode       string            `json:"statusCode"`
	Status           string            `json:"status"`
	PinSet           bool              `json:"pinSet"`
	Expiration       Milliseconds      `json:"expiration"`
	CardDesign       string            `json:"cardDesign"`
	Ucid             string            `json:"ucid"`
	PaymentSystem    string            `json:"paymentSystem"`
	FrozenCard       bool              `json:"frozenCard"`
	HasWrongPins     bool              `json:"hasWrongPins"`
	Value            string            `json:"value"`
	IsEmbossed       bool              `json:"isEmbossed"`
	IsVirtual        bool              `json:"isVirtual"`
	CreationDate     Milliseconds      `json:"creationDate"`
	MultiCardCluster *MultiCardCluster `json:"multiCardCluster,omitempty" gorm:"embedded;embeddedPrefix:multi_card_cluster_"`
	Name             string            `json:"name"`
	IsPaymentDevice  bool              `json:"isPaymentDevice"`
	Primary          bool              `json:"primary"`
	CardIssueType    *string           `json:"cardIssueType,omitempty"`
	SharedResourceId *string           `json:"sharedResourceId,omitempty"`
}

type Loyalty struct {
	ProgramName            string   `json:"programName"`
	ProgramCode            string   `json:"programCode"`
	AccountBackgroundColor string   `json:"accountBackgroundColor"`
	CashbackProgram        bool     `json:"cashbackProgram"`
	CoreGroup              string   `json:"coreGroup"`
	LoyaltyPointsId        uint8    `json:"loyaltyPointsId"`
	AccrualBonuses         *float64 `json:"accrualBonuses,omitempty"`
	LinkedBonuses          *string  `json:"linkedBonuses,omitempty"`
	TotalAvailableBonuses  *float64 `json:"totalAvailableBonuses,omitempty"`
	AvailableBonuses       *float64 `json:"availableBonuses,omitempty"`
}

type AccountShared struct {
	Scopes     []string     `json:"scopes"`
	StartDate  Milliseconds `json:"startDate"`
	OwnerName  string       `json:"ownerName"`
	SharStatus string       `json:"sharStatus"`
}

type Account struct {
	UserPhone string `json:"-" gorm:"index"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	CurrencyCode uint      `json:"-" gorm:"index"`
	Currency     *Currency `json:"currency,omitempty" gorm:"constraint:OnDelete:CASCADE"`

	Id                    string            `json:"id" gorm:"primaryKey"`
	CreditLimit           *MoneyAmount      `json:"creditLimit,omitempty" gorm:"embedded;embeddedPrefix:credit_limit_"`
	MoneyAmount           *MoneyAmount      `json:"moneyAmount,omitempty" gorm:"embedded;embeddedPrefix:money_amount_"`
	DebtBalance           *MoneyAmount      `json:"debtBalance,omitempty" gorm:"embedded;embeddedPrefix:debt_balance_"`
	CurrentMinimalPayment *MoneyAmount      `json:"currentMinimalPayment,omitempty" gorm:"embedded;embeddedPrefix:current_minimal_payment_"`
	ClientUnverifiedFlag  *string           `json:"clientUnverifiedFlag,omitempty"`
	IdentificationState   *string           `json:"identificationState,omitempty"`
	Status                *string           `json:"status,omitempty"`
	EmoneyFlag            *bool             `json:"emoneyFlag,omitempty"`
	NextStatementDate     *Milliseconds     `json:"nextStatementDate,omitempty"`
	DueDate               *Milliseconds     `json:"dueDate,omitempty"`
	Cards                 []Card            `json:"cards,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:AccountId"`
	MultiCardCluster      *MultiCardCluster `json:"multiCardCluster,omitempty" gorm:"embedded;embeddedPrefix:multi_card_cluster_"`
	LoyaltyId             *string           `json:"loyaltyId,omitempty"`
	MoneyPotFlag          *bool             `json:"moneyPotFlag,omitempty"`
	PartNumber            *string           `json:"partNumber,omitempty"`
	PastDueDebt           *MoneyAmount      `json:"pastDueDebt,omitempty" gorm:"embedded;embeddedPrefix:past_due_debt_"`
	Name                  string            `json:"name"`
	AccountType           string            `json:"accountType"`
	Hidden                bool              `json:"hidden"`
	SharedByMeFlag        *bool             `json:"sharedByMeFlag,omitempty"`
	Loyalty               *Loyalty          `json:"loyalty,omitempty" gorm:"embedded;embeddedPrefix:loyalty_"`
	CreationDate          *Milliseconds     `json:"creationDate,omitempty"`
	DebtAmount            *MoneyAmount      `json:"debtAmount,omitempty" gorm:"embedded;embeddedPrefix:debt_amount_"`
	LastStatementDate     *Milliseconds     `json:"lastStatementDate,omitempty"`
	DueColor              *int              `json:"dueColor,omitempty"`
	LinkedAccountNumber   *string           `json:"linkedAccountNumber,omitempty"`
	IsKidsSaving          *bool             `json:"isKidsSaving,omitempty"`
	IsCrowdfunding        *bool             `json:"isCrowdfunding,omitempty"`
	//Shared                *AccountShared    `json:"shared,omitempty"`
}

type Category struct {
	Id   string `json:"id" gorm:"primaryKey"`
	Name string `json:"name" gorm:"index"`
}

type Location struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	Position    int    `json:"-" gorm:"primaryKey"`

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
	Position    int    `json:"-" gorm:"primaryKey"`

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
	Position    int    `json:"-" gorm:"primaryKey"`

	FieldName  string `json:"fieldName"`
	FieldValue string `json:"fieldValue"`
}

type LoyaltyPaymentAmount struct {
	LoyaltyAmount `gorm:"embedded"`
	Price         float64 `json:"price"`
}

type LoyaltyPayment struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	Position    int    `json:"-" gorm:"primaryKey"`

	Amount   LoyaltyPaymentAmount `json:"amount" gorm:"embedded"`
	Status   string               `json:"status"`
	SoldTime *Milliseconds        `json:"soldTime"`
}

type LoyaltyBonusSummary struct {
	Amount float64 `json:"amount"`
}

type Payment struct {
	OperationId string `json:"-" gorm:"primaryKey"`

	SourceIsQr      bool         `json:"sourceIsQr"`
	BankAccountId   string       `json:"bankAccountId"`
	PaymentId       string       `json:"paymentId"`
	ProviderGroupId *string      `json:"providerGroupId,omitempty"`
	PaymentType     string       `json:"paymentType"`
	FeeAmount       *MoneyAmount `json:"feeAmount,omitempty" gorm:"embedded;embeddedPrefix:fee_amount_"`
	ProviderId      string       `json:"providerId"`
	HasPaymentOrder bool         `json:"hasPaymentOrder"`
	Comment         string       `json:"comment"`
	IsQrPayment     bool         `json:"isQrPayment"`
	//FieldsValues       map[string]any `json:"fieldsValues"`
	Repeatable         bool    `json:"repeatable"`
	CardNumber         string  `json:"cardNumber"`
	TemplateId         *string `json:"templateId,omitempty"`
	TemplateIsFavorite *bool   `json:"templateIsFavorite,omitempty"`
}

type Subgroup struct {
	Id   string  `json:"id" gorm:"primaryKey"`
	Name *string `json:"name,omitempty" gorm:"index"`
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

	IsDispute              bool                 `json:"isDispute"`
	IsOffline              bool                 `json:"isOffline"`
	HasStatement           bool                 `json:"hasStatement"`
	IsSuspicious           bool                 `json:"isSuspicious"`
	AuthorizationId        *string              `json:"authorizationId,omitempty"`
	IsInner                bool                 `json:"isInner" gorm:"index"`
	Id                     string               `json:"id" gorm:"primaryKey"`
	Status                 string               `json:"status" gorm:"index"`
	OperationTransferred   bool                 `json:"operationTransferred"`
	IdSourceType           string               `json:"idSourceType"`
	HasShoppingReceipt     *bool                `json:"hasShoppingReceipt,omitempty" gorm:"index"`
	Type                   string               `json:"type" gorm:"index"`
	Locations              []Location           `json:"locations,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	LoyaltyBonus           []LoyaltyBonus       `json:"loyaltyBonus,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	CashbackAmount         MoneyAmount          `json:"cashbackAmount" gorm:"embedded;embeddedPrefix:cashback_"`
	AuthMessage            *string              `json:"authMessage,omitempty"`
	Description            string               `json:"description"`
	IsTemplatable          bool                 `json:"isTemplatable"`
	Cashback               float64              `json:"cashback"`
	Amount                 MoneyAmount          `json:"amount" gorm:"embedded"`
	OperationTime          Milliseconds         `json:"operationTime" gorm:"index"`
	IsHce                  bool                 `json:"isHce"`
	Mcc                    uint                 `json:"mcc"`
	AdditionalInfo         []AdditionalInfo     `json:"additionalInfo,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	VirtualPaymentType     uint8                `json:"virtualPaymentType"`
	Ucid                   *string              `json:"ucid,omitempty"`
	Merchant               *Merchant            `json:"merchant,omitempty" gorm:"embedded;embeddedPrefix:merchant_"`
	Card                   *string              `json:"card,omitempty" gorm:"index"`
	LoyaltyPayment         []LoyaltyPayment     `json:"loyaltyPayment,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	TrancheCreationAllowed bool                 `json:"trancheCreationAllowed"`
	Group                  *string              `json:"group,omitempty"`
	MccString              string               `json:"mccString"`
	CardPresent            bool                 `json:"cardPresent"`
	IsExternalCard         bool                 `json:"isExternalCard"`
	CardNumber             *string              `json:"cardNumber,omitempty"`
	AccountAmount          MoneyAmount          `json:"accountAmount" gorm:"embedded;embeddedPrefix:account_"`
	LoyaltyBonusSummary    *LoyaltyBonusSummary `json:"loyaltyBonusSummary,omitempty" gorm:"embedded;embeddedPrefix:loyalty_bonus_summary_"`
	TypeSerno              *uint                `json:"typeSerno"`
	Payment                *Payment             `json:"payment,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:OperationId"`
	OperationPaymentType   *string              `json:"operationPaymentType,omitempty"`
	DebitingTime           *Milliseconds        `json:"debitingTime,omitempty"`
	PosId                  *string              `json:"posId,omitempty"`
	Subcategory            *string              `json:"subcategory,omitempty" gorm:"index"`
	SenderAgreement        *string              `json:"senderAgreement,omitempty"`
	PointOfSaleId          *uint64              `json:"pointOfSaleId,omitempty"`
	Compensation           *string              `json:"compensation,omitempty"`
	InstallmentStatus      *string              `json:"installmentStatus,omitempty"`
	SenderDetails          *string              `json:"senderDetails,omitempty"`
	PartnerType            *string              `json:"partnerType,omitempty"`
	Nomination             *string              `json:"nomination,omitempty"`
	Message                *string              `json:"message,omitempty"`
	TrancheId              *string              `json:"trancheId,omitempty"`
}

type ReceiptItem struct {
	OperationId string `json:"-" gorm:"primaryKey"`
	Position    int    `json:"-" gorm:"primaryKey"`

	Name     string   `json:"name"`
	Price    float64  `json:"price"`
	Sum      float64  `json:"sum"`
	Quantity float64  `json:"quantity"`
	NdsRate  *uint8   `json:"ndsRate"`
	Nds      *uint8   `json:"nds"`
	Nds10    *float64 `json:"nds10,omitempty"`
	Nds18    *float64 `json:"nds18,omitempty"`
	BrandId  int64    `json:"brand_id"`
	GoodId   int64    `json:"good_id"`
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
