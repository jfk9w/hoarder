package firefly

import (
	"context"

	"github.com/pkg/errors"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type Currencies struct{}

func (s Currencies) TableName() string {
	return "currencies"
}

func (s Currencies) Sync(ctx jobs.Context, db database.DB, client firefly.Invoker) (ss []Interface, errs error) {
	var entities []Currency
	if err := db.WithContext(ctx).
		Where("firefly_id is null").
		Find(&entities).
		Error; ctx.Error(&errs, err, "failed to select pending records") {
		return
	}

	for _, entity := range entities {
		ctx := ctx.With("name", entity.Name)
		fireflyId, err := storeCurrency(ctx, client, entity)
		if ctx.Error(&errs, err, "failed to store currency") {
			continue
		}

		if err := db.WithContext(ctx).
			Table(new(Currency).TableName()).
			Where("code = ?", entity.Code).
			Update("firefly_id", fireflyId).
			Error; ctx.Error(&errs, err, "failed to update firefly id in db") {
			continue
		}
	}

	return
}

func storeCurrency(ctx context.Context, client firefly.Invoker, currency Currency) (string, error) {
	code := currency.Name
	existing, err := getCurrency(ctx, client, code)
	if err != nil {
		return "", errors.Wrap(err, "get currency")
	}

	if existing != nil {
		if !existing.Data.Attributes.Enabled.Value {
			if err := enableCurrency(ctx, client, code); err != nil {
				return "", errors.Wrap(err, "enable currency")
			}
		}

		return existing.Data.ID, nil
	}

	in := &firefly.CurrencyStore{
		Enabled:       firefly.NewOptBool(true),
		Code:          code,
		Name:          code,
		Symbol:        code,
		DecimalPlaces: firefly.NewOptInt32(2),
	}

	out, err := client.StoreCurrency(ctx, in, firefly.StoreCurrencyParams{})
	if err != nil {
		return "", err
	}

	switch out := out.(type) {
	case *firefly.CurrencySingle:
		return out.Data.ID, nil
	case exception:
		return "", exception2error(out)
	default:
		return "", errors.Errorf("%s", out)
	}
}

func enableCurrency(ctx context.Context, client firefly.Invoker, code string) error {
	in := &firefly.CurrencyUpdate{
		Enabled: firefly.NewOptBool(true),
	}

	out, err := client.UpdateCurrency(ctx, in, firefly.UpdateCurrencyParams{Code: code})
	if err != nil {
		return err
	}

	switch out := out.(type) {
	case *firefly.CurrencySingle:
		return nil
	case exception:
		return exception2error(out)
	default:
		return errors.Errorf("%s", out)
	}
}

func getCurrency(ctx context.Context, client firefly.Invoker, code string) (*firefly.CurrencySingle, error) {
	out, err := client.GetCurrency(ctx, firefly.GetCurrencyParams{Code: code})
	if err != nil {
		return nil, err
	}

	switch out := out.(type) {
	case *firefly.CurrencySingle:
		return out, nil
	case *firefly.NotFound:
		return nil, nil
	case exception:
		return nil, exception2error(out)
	default:
		return nil, errors.Errorf("%s", out)
	}
}
