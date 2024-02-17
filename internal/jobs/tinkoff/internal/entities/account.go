package entities

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

type AccountCreditLimit struct {
	CreditLimitCurrencyCode uint     `json:"-" gorm:"index"`
	CreditLimitCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	CreditLimitValue float64 `json:"value"`
}

type AccountMoneyAmount struct {
	MoneyAmountCurrencyCode uint     `json:"-" gorm:"index"`
	MoneyAmountCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	MoneyAmountValue float64 `json:"value"`
}

type AccountDebtBalance struct {
	DebtBalanceCurrencyCode uint     `json:"-" gorm:"index"`
	DebtBalanceCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	DebtBalanceValue float64 `json:"value"`
}

type AccountCurrentMinimalPayment struct {
	CurrentMinimalPaymentCurrencyCode uint     `json:"-" gorm:"index"`
	CurrentMinimalPaymentCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	CurrentMinimalPaymentValue float64 `json:"value"`
}

type AccountPastDueDebt struct {
	PastDueDebtCurrencyCode uint     `json:"-" gorm:"index"`
	PastDueDebtCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	PastDueDebtValue float64 `json:"value"`
}

type AccountDebtAmount struct {
	DebtAmountCurrencyCode uint     `json:"-" gorm:"index"`
	DebtAmountCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	DebtAmountValue float64 `json:"value"`
}

type Account struct {
	UserPhone string `json:"-" gorm:"index"`
	User      User   `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	CurrencyCode *uint     `json:"-" gorm:"index"`
	Currency     *Currency `json:"currency,omitempty" gorm:"constraint:OnDelete:CASCADE"`

	AccountRequisites *AccountRequisites `json:"-"`

	Deleted bool `json:"-" gorm:"index"`

	Id                    string                        `json:"id" gorm:"primaryKey"`
	CreditLimit           *AccountCreditLimit           `json:"creditLimit,omitempty" gorm:"embedded"`
	MoneyAmount           *AccountMoneyAmount           `json:"moneyAmount,omitempty" gorm:"embedded"`
	DebtBalance           *AccountDebtBalance           `json:"debtBalance,omitempty" gorm:"embedded"`
	CurrentMinimalPayment *AccountCurrentMinimalPayment `json:"currentMinimalPayment,omitempty" gorm:"embedded"`
	ClientUnverifiedFlag  *string                       `json:"clientUnverifiedFlag,omitempty"`
	IdentificationState   *string                       `json:"identificationState,omitempty"`
	Status                *string                       `json:"status,omitempty"`
	EmoneyFlag            *bool                         `json:"emoneyFlag,omitempty"`
	NextStatementDate     *Milliseconds                 `json:"nextStatementDate,omitempty"`
	DueDate               *Milliseconds                 `json:"dueDate,omitempty"`
	Cards                 []Card                        `json:"cards,omitempty" gorm:"constraint:OnDelete:CASCADE;foreignKey:AccountId"`
	MultiCardCluster      *MultiCardCluster             `json:"multiCardCluster,omitempty" gorm:"embedded;embeddedPrefix:multi_card_cluster_"`
	LoyaltyId             *string                       `json:"loyaltyId,omitempty"`
	MoneyPotFlag          *bool                         `json:"moneyPotFlag,omitempty"`
	PartNumber            *string                       `json:"partNumber,omitempty"`
	PastDueDebt           *AccountPastDueDebt           `json:"pastDueDebt,omitempty" gorm:"embedded"`
	Name                  string                        `json:"name"`
	AccountType           string                        `json:"accountType"`
	Hidden                bool                          `json:"hidden"`
	SharedByMeFlag        *bool                         `json:"sharedByMeFlag,omitempty"`
	Loyalty               *Loyalty                      `json:"loyalty,omitempty" gorm:"embedded;embeddedPrefix:loyalty_"`
	CreationDate          *Milliseconds                 `json:"creationDate,omitempty"`
	DebtAmount            *AccountDebtAmount            `json:"debtAmount,omitempty" gorm:"embedded"`
	LastStatementDate     *Milliseconds                 `json:"lastStatementDate,omitempty"`
	DueColor              *int                          `json:"dueColor,omitempty"`
	LinkedAccountNumber   *string                       `json:"linkedAccountNumber,omitempty"`
	IsKidsSaving          *bool                         `json:"isKidsSaving,omitempty"`
	IsCrowdfunding        *bool                         `json:"isCrowdfunding,omitempty"`
	//Shared                *AccountShared    `json:"shared,omitempty"`

	FireflyId *string `json:"-" gorm:"<-:false;index"`
}

func (a Account) TableName() string {
	return "accounts"
}

type AccountRequisites struct {
	AccountId string `json:"-" gorm:"primaryKey"`

	CardImage                  string `json:"cardImage"`
	CardLine1                  string `json:"cardLine1"`
	CardLine2                  string `json:"cardLine2"`
	Recipient                  string `json:"recipient"`
	BeneficiaryInfo            string `json:"beneficiaryInfo"`
	BeneficiaryBank            string `json:"beneficiaryBank"`
	RecipientExternalAccount   string `json:"recipientExternalAccount"`
	CorrespondentAccountNumber string `json:"correspondentAccountNumber"`
	BankBik                    string `json:"bankBik"`
	Name                       string `json:"name"`
	Inn                        string `json:"inn"`
	Kpp                        string `json:"kpp"`
}

func (ar AccountRequisites) TableName() string {
	return "account_requisites"
}
