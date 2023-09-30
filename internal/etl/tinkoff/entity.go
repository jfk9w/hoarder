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
	Code    uint   `json:"code" gorm:"primaryKey;autoIncrement:false"`
	Name    string `json:"name" gorm:"index"`
	StrCode string `json:"strCode"`
}

type MoneyAmount struct {
	CurrencyCode uint     `json:"-" gorm:"index"`
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

	Deleted bool `json:"-" gorm:"index"`

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
	DebitingTime           *Milliseconds        `json:"debitingTime,omitempty" gorm:"index"`
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

type StatementPeriod struct {
	Start Milliseconds `json:"start"`
	End   Milliseconds `json:"end"`
}

type Statement struct {
	AccountId string  `json:"-" gorm:"index"`
	Account   Account `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	OverdraftFee           MoneyAmount     `json:"overdraftFee" gorm:"embedded;embeddedPrefix:overdraft_fee_"`
	Expense                MoneyAmount     `json:"expense" gorm:"embedded;embeddedPrefix:expense_"`
	OverLimitDebt          MoneyAmount     `json:"overLimitDebt" gorm:"embedded;embeddedPrefix:over_limit_debt_"`
	PeriodEndBalance       MoneyAmount     `json:"periodEndBalance" gorm:"embedded;embeddedPrefix:period_end_balance_"`
	ArrestAmount           MoneyAmount     `json:"arrestAmount" gorm:"embedded;embeddedPrefix:arrest_amount_"`
	OtherBonus             MoneyAmount     `json:"otherBonus" gorm:"embedded;embeddedPrefix:other_bonus_"`
	CreditLimit            MoneyAmount     `json:"creditLimit" gorm:"embedded;embeddedPrefix:credit_limit_"`
	TranchesMonthlyPayment *MoneyAmount    `json:"tranchesMonthlyPayment,omitempty" gorm:"embedded;embeddedPrefix:tranches_monthly_payment_"`
	BilledDebt             MoneyAmount     `json:"billedDebt" gorm:"embedded;embeddedPrefix:billed_debt_"`
	Cashback               MoneyAmount     `json:"cashback" gorm:"embedded;embeddedPrefix:cashback_"`
	Balance                MoneyAmount     `json:"balance" gorm:"embedded;embeddedPrefix:balance_"`
	HighCashback           MoneyAmount     `json:"highCashback" gorm:"embedded;embeddedPrefix:high_cashback_"`
	PeriodStartBalance     MoneyAmount     `json:"periodStartBalance" gorm:"embedded;embeddedPrefix:period_start_balance_"`
	LowCashback            MoneyAmount     `json:"lowCashback" gorm:"embedded;embeddedPrefix:low_cashback_"`
	AvailableLimit         MoneyAmount     `json:"availableLimit" gorm:"embedded;embeddedPrefix:available_limit_"`
	Id                     string          `json:"id" gorm:"primaryKey"`
	InterestBonus          MoneyAmount     `json:"interestBonus" gorm:"embedded;embeddedPrefix:interest_bonus_"`
	Interest               MoneyAmount     `json:"interest" gorm:"embedded;embeddedPrefix:interest_"`
	Date                   Milliseconds    `json:"date" gorm:"index"`
	Income                 MoneyAmount     `json:"income" gorm:"embedded;embeddedPrefix:income_"`
	CreditBonus            MoneyAmount     `json:"creditBonus" gorm:"embedded;embeddedPrefix:credit_bonus_"`
	LastPaymentDate        *Milliseconds   `json:"lastPaymentDate,omitempty"`
	OtherCashback          MoneyAmount     `json:"otherCashback" gorm:"embedded;embeddedPrefix:other_cashback_"`
	MinimalPaymentAmount   *MoneyAmount    `json:"minimalPaymentAmount,omitempty" gorm:"embedded;embeddedPrefix:minimal_payment_amount_"`
	PastDueDebt            *MoneyAmount    `json:"pastDueDebt,omitempty" gorm:"embedded;embeddedPrefix:past_due_debt_"`
	Period                 StatementPeriod `json:"period" gorm:"embedded;embeddedPrefix:period_"`
	NoOverdue              *bool           `json:"noOverdue,omitempty"`
	Repaid                 *string         `json:"repaid,omitempty"`
}

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
	Logo         string `json:"logo"`
	ExternalCode string `json:"externalCode"`
	ExternalId   string `json:"externalId"`
	Id           string `json:"id" gorm:"primaryKey"`
	Percent      uint   `json:"percent"`
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

type InvestOperationType struct {
	Deleted bool `json:"-" gorm:"index"`

	Category      string `json:"category"`
	OperationName string `json:"operationName"`
	OperationType string `json:"operationType" gorm:"primaryKey"`
}

type InvestAmount struct {
	Currency string  `json:"currency"`
	Value    float64 `json:"value"`
}

type InvestTotals struct {
	ExpectedYield                InvestAmount `json:"expectedYield" gorm:"embedded;embeddedPrefix:expected_yield_"`
	ExpectedYieldRelative        float64      `json:"expectedYieldRelative"`
	ExpectedYieldPerDay          InvestAmount `json:"expectedYieldPerDay" gorm:"embedded;embeddedPrefix:expected_yield_per_day_"`
	ExpectedYieldPerDayRelative  float64      `json:"expectedYieldPerDayRelative"`
	ExpectedAverageYield         InvestAmount `json:"expectedAverageYield" gorm:"embedded;embeddedPrefix:expected_average_yield_"`
	ExpectedAverageYieldRelative float64      `json:"expectedAverageYieldRelative"`
	TotalAmount                  InvestAmount `json:"totalAmount" gorm:"embedded;embeddedPrefix:total_amount_"`
}

type InvestAccount struct {
	UserPhone string `json:"-" gorm:"index"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	Deleted bool `json:"-" gorm:"index"`

	Id            string `json:"brokerAccountId" gorm:"primaryKey"`
	Type          string `json:"brokerAccountType"`
	Name          string `json:"name"`
	OpenedDate    Date   `json:"openedDate"`
	Order         int    `json:"order"`
	Status        string `json:"status"`
	IsVisible     bool   `json:"isVisible"`
	Organization  string `json:"organization"`
	BuyByDefault  bool   `json:"buyByDefault"`
	MarginEnabled bool   `json:"marginEnabled"`
	AutoApp       bool   `json:"autoApp"`

	InvestTotals `gorm:"embedded"`
}

type Trade struct {
	InvestOperationInternalId string `json:"-" gorm:"primaryKey"`
	DbIdx                     int    `json:"dbIdx" gorm:"primaryKey"`

	Date          DateTimeMilliOffset `json:"date"`
	Num           string              `json:"num"`
	Price         InvestAmount        `json:"price" gorm:"embedded;embeddedPrefix:price_"`
	Quantity      int                 `json:"quantity"`
	Yield         *InvestAmount       `json:"yield,omitempty" gorm:"embedded;embeddedPrefix:yield_"`
	YieldRelative *float64            `json:"yieldRelative,omitempty"`
}

type TradesInfo struct {
	Trades     []Trade `json:"trades" gorm:"constraint:OnDelete:CASCADE;foreignKey:InvestOperationInternalId"`
	TradesSize int     `json:"tradesSize"`
}

type InvestChildOperation struct {
	InvestOperationInternalId string `json:"-" gorm:"primaryKey"`
	DbIdx                     int    `json:"dbIdx" gorm:"primaryKey"`

	Currency       string       `json:"currency"`
	Id             string       `json:"id"`
	InstrumentType string       `json:"instrumentType"`
	InstrumentUid  string       `json:"instrumentUid"`
	LogoName       string       `json:"logoName"`
	Payment        InvestAmount `json:"payment" gorm:"embedded;embeddedPrefix:payment_"`
	ShowName       string       `json:"showName"`
	Ticker         string       `json:"ticker"`
	Type           string       `json:"type"`
	Value          float64      `json:"value"`
}

type InvestOperation struct {
	InvestAccountId string        `json:"brokerAccountId" gorm:"index"`
	InvestAccount   InvestAccount `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	AssetUid                      *string                `json:"assetUid,omitempty"`
	BestExecuted                  bool                   `json:"bestExecuted"`
	ClassCode                     *string                `json:"classCode,omitempty"`
	Cursor                        string                 `json:"cursor"`
	Date                          DateTimeMilliOffset    `json:"date"`
	Description                   string                 `json:"description"`
	Id                            *string                `json:"id,omitempty"`
	InstrumentType                *string                `json:"instrumentType,omitempty"`
	InstrumentUid                 *string                `json:"instrumentUid,omitempty"`
	InternalId                    string                 `json:"internalId" gorm:"primaryKey"`
	IsBlockedTradeClearingAccount *bool                  `json:"isBlockedTradeClearingAccount,omitempty"`
	Isin                          *string                `json:"isin,omitempty"`
	LogoName                      *string                `json:"logoName,omitempty"`
	Name                          *string                `json:"name,omitempty"`
	Payment                       InvestAmount           `json:"payment" gorm:"embedded;embeddedPrefix:payment_"`
	PaymentEur                    InvestAmount           `json:"paymentEur" gorm:"embedded;embeddedPrefix:payment_eur_"`
	PaymentRub                    InvestAmount           `json:"paymentRub" gorm:"embedded;embeddedPrefix:payment_rub_"`
	PaymentUsd                    InvestAmount           `json:"paymentUsd" gorm:"embedded;embeddedPrefix:payment_usd_"`
	PositionUid                   *string                `json:"positionUid,omitempty"`
	ShortDescription              *string                `json:"shortDescription,omitempty"`
	ShowName                      *string                `json:"showName,omitempty"`
	Status                        string                 `json:"status"`
	TextColor                     *string                `json:"textColor,omitempty"`
	Ticker                        *string                `json:"ticker,omitempty"`
	Type                          string                 `json:"type"`
	AccountId                     *string                `json:"accountId,omitempty"`
	DoneRest                      *int                   `json:"doneRest,omitempty"`
	Price                         *InvestAmount          `json:"price,omitempty" gorm:"embedded;embeddedPrefix:price_"`
	Quantity                      *int                   `json:"quantity,omitempty"`
	TradesInfo                    *TradesInfo            `json:"tradesInfo,omitempty" gorm:"embedded"`
	ParentOperationId             *string                `json:"parentOperationId,omitempty"`
	ChildOperations               []InvestChildOperation `json:"childOperations,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:InvestOperationInternalId"`
	Commission                    *InvestAmount          `json:"commission,omitempty" gorm:"embedded;embeddedPrefix:commission_"`
	Yield                         *InvestAmount          `json:"yield,omitempty" gorm:"embedded;embeddedPrefix:yield_"`
	YieldRelative                 *float64               `json:"yieldRelative,omitempty"`
	CancelReason                  *string                `json:"cancelReason,omitempty"`
	QuantityRest                  *int                   `json:"quantityRest,omitempty"`
	WithdrawDateTime              *DateTime              `json:"withdrawDateTime,omitempty"`
}
