package firefly

import (
	"context"

	"github.com/pkg/errors"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type Categories struct{}

func (s Categories) TableName() string {
	return "categories"
}

func (s Categories) Sync(ctx jobs.Context, db database.DB, client firefly.Invoker) (ss []Interface, errs error) {
	var entities []SpendingCategory
	if err := db.WithContext(ctx).
		Where("firefly_id is null").
		Find(&entities).
		Error; ctx.Error(&errs, err, "failed to select pending records") {
		return
	}

	for _, entity := range entities {
		ctx := ctx.With("name", entity.Name)
		fireflyId, err := storeCategory(ctx, client, entity)
		if ctx.Error(&errs, err, "failed to store category") {
			continue
		}

		if err := db.WithContext(ctx).
			Table(new(SpendingCategory).TableName()).
			Where("id = ?", entity.Id).
			Update("firefly_id", fireflyId).
			Error; ctx.Error(&errs, err, "failed to update firefly id in db") {
			continue
		}
	}

	return
}

func storeCategory(ctx context.Context, client firefly.Invoker, category SpendingCategory) (string, error) {
	in := &firefly.Category{
		Name: category.Name,
	}

	out, err := client.StoreCategory(ctx, in, firefly.StoreCategoryParams{})
	if err != nil {
		return "", err
	}

	switch out := out.(type) {
	case *firefly.CategorySingle:
		return out.Data.ID, nil
	case exception:
		return "", exception2error(out)
	default:
		return "", errors.Errorf("%s", out)
	}
}
