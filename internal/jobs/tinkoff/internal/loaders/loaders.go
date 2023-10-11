package loaders

import (
	"context"

	"github.com/jfk9w-go/tinkoff-api"
	"gorm.io/gorm/schema"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
)

type Client interface {
	AccountsLightIb(ctx context.Context) (tinkoff.AccountsLightIbOut, error)
	AccountRequisites(ctx context.Context, in *tinkoff.AccountRequisitesIn) (*tinkoff.AccountRequisitesOut, error)
	Statements(ctx context.Context, in *tinkoff.StatementsIn) (tinkoff.StatementsOut, error)
	Operations(ctx context.Context, in *tinkoff.OperationsIn) (tinkoff.OperationsOut, error)
	ShoppingReceipt(ctx context.Context, in *tinkoff.ShoppingReceiptIn) (*tinkoff.ShoppingReceiptOut, error)
	ClientOfferEssences(ctx context.Context) (tinkoff.ClientOfferEssencesOut, error)
	InvestOperationTypes(ctx context.Context) (*tinkoff.InvestOperationTypesOut, error)
	InvestAccounts(ctx context.Context, in *tinkoff.InvestAccountsIn) (*tinkoff.InvestAccountsOut, error)
	InvestOperations(ctx context.Context, in *tinkoff.InvestOperationsIn) (*tinkoff.InvestOperationsOut, error)
}

type Interface interface {
	schema.Tabler
	Load(ctx jobs.Context, client Client, db database.DB) ([]Interface, error)
}
