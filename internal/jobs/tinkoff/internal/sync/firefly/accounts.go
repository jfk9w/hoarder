package firefly

import (
	"context"
	"fmt"

	"github.com/AlekSi/pointer"
	"github.com/pkg/errors"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type accounts struct {
	phone     string
	batchSize int
}

func (s accounts) TableName() string {
	return "accounts"
}

func (s accounts) Sync(ctx jobs.Context, db database.DB, client firefly.Invoker) (ss []Interface, errs error) {
	ctx = ctx.With("phone", s.phone)

	var entities []Account
	if err := db.WithContext(ctx).
		Where("user_phone = ?", s.phone).
		Preload("Currency").
		Find(&entities).
		Error; ctx.Error(&errs, err, "failed to select records") {
		return
	}

	for _, entity := range entities {
		ctx := ctx.With("id", entity.Id)
		if entity.FireflyId != nil {
			err := updateAccount(ctx, client, entity)
			if !ctx.Error(&errs, err, "failed to update account") {
				ss = append(ss, transactions{accountId: entity.Id, batchSize: s.batchSize})
			}

			continue
		}

		fireflyId, err := storeAccount(ctx, client, entity)
		if ctx.Error(&errs, err, "failed to store account") {
			continue
		}

		if err := db.WithContext(ctx).
			Table(new(Account).TableName()).
			Where("id = ?", entity.Id).
			Update("firefly_id", fireflyId).
			Error; ctx.Error(&errs, err, "failed to update firefly id in db") {
			continue
		}

		ss = append(ss, transactions{accountId: entity.Id, batchSize: s.batchSize})
	}

	return
}

func updateAccount(ctx context.Context, client firefly.Invoker, account Account) error {
	in := &firefly.AccountUpdate{
		Name:   getAccountName(account),
		Active: firefly.NewOptBool(!account.Deleted),
	}

	out, err := client.UpdateAccount(ctx, in, firefly.UpdateAccountParams{ID: pointer.Get(account.FireflyId)})
	if err != nil {
		return err
	}

	switch out := out.(type) {
	case *firefly.AccountSingle:
		return nil
	case exception:
		return exception2error(out)
	default:
		return errors.Errorf("%s", out)
	}
}

func storeAccount(ctx context.Context, client firefly.Invoker, account Account) (string, error) {
	in := &firefly.AccountStore{
		Name:          getAccountName(account),
		Type:          firefly.ShortAccountTypePropertyAsset,
		AccountNumber: firefly.NewOptNilString(account.Id),
		Active:        firefly.NewOptBool(!account.Deleted),
	}

	if currency := account.Currency; currency != nil {
		if fireflyId := currency.FireflyId; fireflyId != nil {
			in.CurrencyID = firefly.NewOptString(pointer.Get(currency.FireflyId))
		} else {
			in.CurrencyCode = firefly.NewOptString(currency.Name)
		}
	}

	switch account.AccountType {
	case "Current":
		in.AccountRole = firefly.NewOptNilAccountRoleProperty(firefly.AccountRolePropertyDefaultAsset)
	case "Saving":
		in.AccountRole = firefly.NewOptNilAccountRoleProperty(firefly.AccountRolePropertySavingAsset)
	case "Credit":
		in.AccountRole = firefly.NewOptNilAccountRoleProperty(firefly.AccountRolePropertyCcAsset)
		in.CreditCardType = firefly.NewOptNilCreditCardType(firefly.CreditCardTypeMonthlyFull)
		if dueDate := account.DueDate; dueDate != nil {
			in.MonthlyPaymentDate = firefly.NewOptNilDateTime(dueDate.Time())
		}
	}

	out, err := client.StoreAccount(ctx, in, firefly.StoreAccountParams{})
	if err != nil {
		return "", err
	}

	switch out := out.(type) {
	case *firefly.AccountSingle:
		return out.Data.ID, nil
	case exception:
		return "", exception2error(out)
	default:
		return "", errors.Errorf("%s", out)
	}
}

func getAccountName(account Account) string {
	return fmt.Sprintf("%s (%s)", account.Name, account.Id)
}
