package entities

type InvestOperationType struct {
	Deleted bool `json:"-" gorm:"index"`

	Category      string `json:"category"`
	OperationName string `json:"operationName"`
	OperationType string `json:"operationType" gorm:"primaryKey"`
}

func (t InvestOperationType) TableName() string {
	return "invest_operation_types"
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

func (a InvestAccount) TableName() string {
	return "invest_accounts"
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

func (o InvestOperation) TableName() string {
	return "invest_operations"
}
