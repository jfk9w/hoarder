package tbank

import (
	. "github.com/jfk9w/hoarder/internal/jobs/tbank/internal/entities"
)

var entities = []any{
	new(User),
	new(Session),
	new(Currency),
	new(Account),
	new(Card),
	new(AccountRequisites),
	new(Statement),
	new(Category),
	new(SpendingCategory),
	new(Brand),
	new(Subgroup),
	new(Operation),
	new(Location),
	new(LoyaltyBonus),
	new(AdditionalInfo),
	new(LoyaltyPayment),
	new(Payment),
	new(PaymentFieldValue),
	new(Receipt),
	new(ReceiptItem),
	new(InvestOperationType),
	new(InvestAccount),
	new(InvestOperation),
	new(Trade),
	new(InvestChildOperation),
	new(ClientOffer),
	new(ClientOfferAccount),
	new(ClientOfferEssence),
	new(ClientOfferEssenceMccCode),
}
