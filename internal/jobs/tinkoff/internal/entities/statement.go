package entities

type StatementPeriod struct {
	Start Milliseconds `json:"start"`
	End   Milliseconds `json:"end"`
}

type StatementOverdraftFee struct {
	OverdraftFeeCurrencyCode uint     `json:"-" gorm:"index"`
	OverdraftFeeCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	OverdraftFeeValue float64 `json:"value"`
}

type StatementExpense struct {
	ExpenseCurrencyCode uint     `json:"-" gorm:"index"`
	ExpenseCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	ExpenseValue float64 `json:"value"`
}

type StatementOverLimitDebt struct {
	OverLimitDebtCurrencyCode uint     `json:"-" gorm:"index"`
	OverLimitDebtCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	OverLimitDebtValue float64 `json:"value"`
}

type StatementPeriodEndBalance struct {
	PeriodEndBalanceCurrencyCode uint     `json:"-" gorm:"index"`
	PeriodEndBalanceCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	PeriodEndBalanceValue float64 `json:"value"`
}

type StatementArrestAmount struct {
	ArrestAmountCurrencyCode uint     `json:"-" gorm:"index"`
	ArrestAmountCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	ArrestAmountValue float64 `json:"value"`
}

type StatementOtherBonus struct {
	OtherBonusCurrencyCode uint     `json:"-" gorm:"index"`
	OtherBonusCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	OtherBonusValue float64 `json:"value"`
}

type StatementCreditLimit struct {
	CreditLimitCurrencyCode uint     `json:"-" gorm:"index"`
	CreditLimitCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	CreditLimitValue float64 `json:"value"`
}

type StatementTranchesMonthlyPayment struct {
	TranchesMonthlyPaymentCurrencyCode uint     `json:"-" gorm:"index"`
	TranchesMonthlyPaymentCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	TranchesMonthlyPaymentValue float64 `json:"value"`
}

type StatementBilledDebt struct {
	BilledDebtCurrencyCode uint     `json:"-" gorm:"index"`
	BilledDebtCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	BilledDebtValue float64 `json:"value"`
}

type StatementCashback struct {
	CashbackCurrencyCode uint     `json:"-" gorm:"index"`
	CashbackCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	CashbackValue float64 `json:"value"`
}

type StatementBalance struct {
	BalanceCurrencyCode uint     `json:"-" gorm:"index"`
	BalanceCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	BalanceValue float64 `json:"value"`
}

type StatementHighCashback struct {
	HighCashbackCurrencyCode uint     `json:"-" gorm:"index"`
	HighCashbackCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	HighCashbackValue float64 `json:"value"`
}

type StatementPeriodStartBalance struct {
	PeriodStartBalanceCurrencyCode uint     `json:"-" gorm:"index"`
	PeriodStartBalanceCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	PeriodStartBalanceValue float64 `json:"value"`
}

type StatementLowCashback struct {
	LowCashbackCurrencyCode uint     `json:"-" gorm:"index"`
	LowCashbackCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	LowCashbackValue float64 `json:"value"`
}

type StatementAvailableLimit struct {
	AvailableLimitCurrencyCode uint     `json:"-" gorm:"index"`
	AvailableLimitCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	AvailableLimitValue float64 `json:"value"`
}

type StatementInterestBonus struct {
	InterestBonusCurrencyCode uint     `json:"-" gorm:"index"`
	InterestBonusCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	InterestBonusValue float64 `json:"value"`
}

type StatementInterest struct {
	InterestCurrencyCode uint     `json:"-" gorm:"index"`
	InterestCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	InterestValue float64 `json:"value"`
}

type StatementIncome struct {
	IncomeCurrencyCode uint     `json:"-" gorm:"index"`
	IncomeCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	IncomeValue float64 `json:"value"`
}

type StatementCreditBonus struct {
	CreditBonusCurrencyCode uint     `json:"-" gorm:"index"`
	CreditBonusCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	CreditBonusValue float64 `json:"value"`
}

type StatementOtherCashback struct {
	OtherCashbackCurrencyCode uint     `json:"-" gorm:"index"`
	OtherCashbackCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	OtherCashbackValue float64 `json:"value"`
}

type StatementMinimalPaymentAmount struct {
	MinimalPaymentAmountCurrencyCode uint     `json:"-" gorm:"index"`
	MinimalPaymentAmountCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	MinimalPaymentAmountValue float64 `json:"value"`
}

type StatementPastDueDebt struct {
	PastDueDebtCurrencyCode uint     `json:"-" gorm:"index"`
	PastDueDebtCurrency     Currency `json:"currency" gorm:"constraint:OnDelete:CASCADE"`

	PastDueDebtValue float64 `json:"value"`
}

type Statement struct {
	AccountId string  `json:"-" gorm:"index"`
	Account   Account `json:"-" gorm:"constraint:OnDelete:CASCADE"`

	OverdraftFee           *StatementOverdraftFee           `json:"overdraftFee,omitempty" gorm:"embedded"`
	Expense                StatementExpense                 `json:"expense" gorm:"embedded"`
	OverLimitDebt          *StatementOverLimitDebt          `json:"overLimitDebt,omitempty" gorm:"embedded"`
	PeriodEndBalance       StatementPeriodEndBalance        `json:"periodEndBalance" gorm:"embedded"`
	ArrestAmount           *StatementArrestAmount           `json:"arrestAmount,omitempty" gorm:"embedded"`
	OtherBonus             *StatementOtherBonus             `json:"otherBonus,omitempty" gorm:"embedded"`
	CreditLimit            *StatementCreditLimit            `json:"creditLimit,omitempty" gorm:"embedded"`
	TranchesMonthlyPayment *StatementTranchesMonthlyPayment `json:"tranchesMonthlyPayment,omitempty" gorm:"embedded"`
	BilledDebt             *StatementBilledDebt             `json:"billedDebt,omitempty" gorm:"embedded"`
	Cashback               StatementCashback                `json:"cashback" gorm:"embedded"`
	Balance                StatementBalance                 `json:"balance" gorm:"embedded"`
	HighCashback           *StatementHighCashback           `json:"highCashback,omitempty" gorm:"embedded"`
	PeriodStartBalance     StatementPeriodStartBalance      `json:"periodStartBalance" gorm:"embedded"`
	LowCashback            *StatementLowCashback            `json:"lowCashback,omitempty" gorm:"embedded"`
	AvailableLimit         *StatementAvailableLimit         `json:"availableLimit,omitempty" gorm:"embedded"`
	Id                     string                           `json:"id" gorm:"primaryKey"`
	InterestBonus          *StatementInterestBonus          `json:"interestBonus,omitempty" gorm:"embedded"`
	Interest               StatementInterest                `json:"interest" gorm:"embedded"`
	Date                   Milliseconds                     `json:"date" gorm:"index"`
	Income                 StatementIncome                  `json:"income" gorm:"embedded"`
	CreditBonus            *StatementCreditBonus            `json:"creditBonus,omitempty" gorm:"embedded"`
	LastPaymentDate        *Milliseconds                    `json:"lastPaymentDate,omitempty"`
	OtherCashback          *StatementOtherCashback          `json:"otherCashback,omitempty" gorm:"embedded"`
	MinimalPaymentAmount   *StatementMinimalPaymentAmount   `json:"minimalPaymentAmount,omitempty" gorm:"embedded"`
	PastDueDebt            *StatementPastDueDebt            `json:"pastDueDebt,omitempty" gorm:"embedded"`
	Period                 StatementPeriod                  `json:"period" gorm:"embedded;embeddedPrefix:period_"`
	NoOverdue              *bool                            `json:"noOverdue,omitempty"`
	Repaid                 *string                          `json:"repaid,omitempty"`
}

func (s Statement) TableName() string {
	return "statements"
}
