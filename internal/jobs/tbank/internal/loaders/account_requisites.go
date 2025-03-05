package loaders

import (
	tbank "github.com/jfk9w-go/tbank-api"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tbank/internal/entities"
)

type accountRequisites struct {
	accountId string
}

func (l accountRequisites) TableName() string {
	return new(AccountRequisites).TableName()
}

func (l accountRequisites) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	ctx = ctx.With("account_id", l.accountId)

	out, err := client.AccountRequisites(ctx, &tbank.AccountRequisitesIn{Account: l.accountId})
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	if out == nil {
		return
	}

	entity, err := database.ToViaJSON[AccountRequisites](out)
	if ctx.Error(&errs, err, "entity conversion failed") {
		return
	}

	entity.AccountId = l.accountId

	if err := db.WithContext(ctx).
		Upsert(entity).
		Error; ctx.Error(&errs, err, "failed to update entity in db") {
		return
	}

	ctx.Info("updated entity in db")
	return
}
