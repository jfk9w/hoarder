package loaders

import (
	"context"

	tbank "github.com/jfk9w-go/tbank-api"
	"gorm.io/gorm/schema"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
)

type Client interface {
	AccountsLightIb(ctx context.Context) (tbank.AccountsLightIbOut, error)
	AccountRequisites(ctx context.Context, in *tbank.AccountRequisitesIn) (*tbank.AccountRequisitesOut, error)
	Statements(ctx context.Context, in *tbank.StatementsIn) (tbank.StatementsOut, error)
	Operations(ctx context.Context, in *tbank.OperationsIn) (tbank.OperationsOut, error)
	ShoppingReceipt(ctx context.Context, in *tbank.ShoppingReceiptIn) (*tbank.ShoppingReceiptOut, error)
	ClientOfferEssences(ctx context.Context) (tbank.ClientOfferEssencesOut, error)
	InvestOperationTypes(ctx context.Context) (*tbank.InvestOperationTypesOut, error)
	InvestAccounts(ctx context.Context, in *tbank.InvestAccountsIn) (*tbank.InvestAccountsOut, error)
	InvestOperations(ctx context.Context, in *tbank.InvestOperationsIn) (*tbank.InvestOperationsOut, error)
	Ping(ctx context.Context)
}

type Interface interface {
	schema.Tabler
	Load(ctx jobs.Context, client Client, db database.DB) ([]Interface, error)
}
