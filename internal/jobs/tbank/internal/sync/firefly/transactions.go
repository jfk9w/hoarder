package firefly

import (
	"context"
	"time"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tbank/internal/entities"
)

type transactionQueryRow struct {
	OperationId                     string
	OperationTime                   time.Time
	DebitingTime                    time.Time
	Description                     string
	FireflyCategoryId               string
	FireflyCurrencyId               string
	Amount                          string
	FireflyForeignCurrencyId        string
	ForeignAmount                   string
	SourceOperationId               *string
	FireflySourceTransactionId      *string
	FireflySourceAccountId          *string
	DestinationOperationId          *string
	FireflyDestinationTransactionId *string
	FireflyDestinationAccountId     *string
}

type transactions struct {
	accountId string
	batchSize int
}

func (s transactions) TableName() string {
	return "transactions"
}

func (s transactions) Sync(ctx jobs.Context, db database.DB, client firefly.Invoker) ([]Interface, error) {
	ctx = ctx.With("account_id", s.accountId)
	return nil, jobs.Batch[int]{
		Key:  "offset",
		Size: s.batchSize,
	}.Run(ctx, transactionsBatch{
		db:        db,
		client:    client,
		accountId: s.accountId,
	}.sync)
}

type transactionsBatch struct {
	db        database.DB
	client    firefly.Invoker
	accountId string
}

func (s transactionsBatch) sync(ctx jobs.Context, offset int, limit int) (nextOffset *int, errs error) {
	var rows []transactionQueryRow
	if err := s.db.WithContext(ctx).
		Raw(transactionQuerySQL, s.accountId, limit).
		Scan(&rows).
		Error; ctx.Error(&errs, err, "failed to query operations") {
		return
	}

	for _, row := range rows {
		ctx := ctx.With("operation_id", row.OperationId)
		if row.SourceOperationId != nil && row.DestinationOperationId != nil &&
			((row.FireflySourceTransactionId == nil) != (row.FireflyDestinationTransactionId == nil)) {
			var (
				transactionId *string
				operationId   *string
			)

			if row.FireflySourceTransactionId != nil {
				transactionId = row.FireflySourceTransactionId
				operationId = row.SourceOperationId
			} else {
				transactionId = row.FireflyDestinationTransactionId
				operationId = row.DestinationOperationId
			}

			if err := deleteTransaction(ctx, s.client, *transactionId); ctx.Error(&errs, err, "failed to delete transaction") {
				continue
			}

			if err := s.db.WithContext(ctx).
				Table(new(Operation).TableName()).
				Where("id = ?", *operationId).
				Update("firefly_id", nil).
				Error; ctx.Error(&errs, err, "failed to delete firefly id from db") {
				continue
			}

			row.FireflySourceTransactionId = nil
			row.FireflyDestinationTransactionId = nil
		}

		if row.FireflySourceTransactionId != nil || row.FireflyDestinationTransactionId != nil {
			continue
		}

		fireflyId, err := storeTransaction(ctx, s.client, &row)
		if ctx.Error(&errs, err, "failed to store transaction") {
			continue
		}

		if err := s.db.WithContext(ctx).
			Table(new(Operation).TableName()).
			Where("id in ?", []any{row.SourceOperationId, row.DestinationOperationId}).
			Update("firefly_id", fireflyId).
			Error; ctx.Error(&errs, err, "failed to update firefly id in db") {
			continue
		}
	}

	if len(rows) == limit {
		nextOffset = pointer.To(offset + limit)
	}

	return
}

type transaction interface {
	SetProcessDate(firefly.OptNilDateTime)
	SetCategoryID(firefly.OptNilString)
	SetCurrencyID(firefly.OptNilString)
	SetForeignCurrencyID(firefly.OptNilString)
	SetForeignAmount(firefly.OptNilString)
	SetSourceID(firefly.OptNilString)
	SetDestinationID(firefly.OptNilString)
}

func deleteTransaction(ctx context.Context, client firefly.Invoker, transactionId string) error {
	out, err := client.DeleteTransaction(ctx, firefly.DeleteTransactionParams{ID: transactionId})
	if err != nil {
		return err
	}

	switch out := out.(type) {
	case *firefly.DeleteTransactionNoContent, *firefly.NotFound:
		return nil
	case exception:
		return exception2error(out)
	default:
		return errors.Errorf("%s", out)
	}
}

func storeTransaction(ctx context.Context, client firefly.Invoker, row *transactionQueryRow) (string, error) {
	var transaction firefly.TransactionSplitStore
	transactionType := setTransactionFields(row, &transaction)
	transaction.SetType(transactionType)
	transaction.SetDate(row.OperationTime)
	transaction.SetDescription(row.Description)
	transaction.SetAmount(row.Amount)

	in := &firefly.TransactionStore{
		Transactions: []firefly.TransactionSplitStore{
			transaction,
		},
	}

	out, err := client.StoreTransaction(ctx, in, firefly.StoreTransactionParams{})
	if err != nil {
		return "", err
	}

	switch out := out.(type) {
	case *firefly.TransactionSingle:
		return out.Data.ID, nil
	case exception:
		return "", exception2error(out)
	default:
		return "", errors.Errorf("%s", out)
	}
}

func setTransactionFields(row *transactionQueryRow, transaction transaction) firefly.TransactionTypeProperty {
	transaction.SetProcessDate(firefly.NewOptNilDateTime(row.OperationTime))
	transaction.SetCategoryID(firefly.NewOptNilString(row.FireflyCategoryId))

	var transactionType firefly.TransactionTypeProperty
	if row.FireflySourceAccountId != nil {
		transactionType = firefly.TransactionTypePropertyWithdrawal
		transaction.SetSourceID(firefly.NewOptNilString(*row.FireflySourceAccountId))
	} else {
		transactionType = firefly.TransactionTypePropertyDeposit
		row.FireflyCurrencyId, row.FireflyForeignCurrencyId = row.FireflyForeignCurrencyId, row.FireflyCurrencyId
		row.Amount, row.ForeignAmount = row.ForeignAmount, row.Amount
	}

	if row.FireflyDestinationAccountId != nil {
		if transactionType == firefly.TransactionTypePropertyWithdrawal {
			transactionType = firefly.TransactionTypePropertyTransfer
		}

		transaction.SetDestinationID(firefly.NewOptNilString(*row.FireflyDestinationAccountId))
	}

	transaction.SetCurrencyID(firefly.NewOptNilString(row.FireflyCurrencyId))
	if row.FireflyCurrencyId != row.FireflyForeignCurrencyId || transactionType == firefly.TransactionTypePropertyTransfer {
		transaction.SetForeignCurrencyID(firefly.NewOptNilString(row.FireflyForeignCurrencyId))
		transaction.SetForeignAmount(firefly.NewOptNilString(row.ForeignAmount))
	}

	return transactionType
}

const transactionQuerySQL = `
with o as (select *, row_number() over (order by operation_time, id::bigint) as num
           from operations
           where status = 'OK'
             and debiting_time is not null)
select coalesce(lo.id, ro.id)                             as operation_id,
       coalesce(lo.firefly_id, ro.firefly_id)             as firefly_id,
       coalesce(lo.operation_time, ro.operation_time)     as operation_time,
       coalesce(lo.debiting_time, ro.debiting_time)       as debiting_time,
       coalesce(lo.description, ro.description)           as description,
       sc.firefly_id                                      as firefly_category_id,
       lc.firefly_id                                      as firefly_currency_id,
       coalesce(lo.account_value, ro.value)::text         as amount,
       rc.firefly_id                                      as firefly_foreign_currency_id,
       coalesce(ro.account_value, lo.value)::text         as foreign_amount,
       case when la.firefly_id is not null then lo.id end as source_operation_id,
       lo.firefly_id                                      as firefly_source_transaction_id,
       la.firefly_id                                      as firefly_source_account_id,
       case when ra.firefly_id is not null then ro.id end as destination_operation_id,
       ro.firefly_id                                      as firefly_destination_transaction_id,
       ra.firefly_id                                      as firefly_destination_account_id
from o as lo
         full join o as ro on ro.sender_agreement = lo.account_id and
                              (lo.num + 1 = ro.num or lo.operation_time = ro.operation_time) and
                              (lo.value = ro.account_value or lo.account_value = ro.value) and
                              (lo."group" = 'TRANSFER' and ro."group" = 'INCOME')
         inner join spending_categories sc on coalesce(lo.spending_category_id, ro.spending_category_id) = sc.id
         inner join currencies lc on coalesce(lo.account_currency_code, ro.currency_code) = lc.code
         inner join currencies rc on coalesce(ro.account_currency_code, lo.currency_code) = rc.code
         left join accounts la on lo.account_id = la.id
         left join accounts ra on ro.account_id = ra.id
where coalesce(lo.type, 'Debit') = 'Debit'
  and coalesce(ro.type, 'Credit') = 'Credit'
  and ? in (lo.account_id, ro.account_id)
  and (la.firefly_id is not null and lo.firefly_id is null or ra.firefly_id is not null and ro.firefly_id is null)
order by 1::bigint
limit ?
`
