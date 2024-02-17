package loaders

import (
	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type InvestOperationTypes struct {
	BatchSize int
}

func (l InvestOperationTypes) TableName() string {
	return new(InvestOperationType).TableName()
}

func (l InvestOperationTypes) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	out, err := client.InvestOperationTypes(ctx)
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	var (
		uniqueTypes = make(map[string]bool)
		entities    []InvestOperationType
		types       []string
	)

	for _, out := range out.OperationsTypes {
		ctx := ctx.With("operation_type", out.OperationType)

		if uniqueTypes[out.OperationType] {
			ctx.Debug("skipping duplicate")
			continue
		}

		entity, err := database.ToViaJSON[InvestOperationType](out)
		if ctx.Error(&errs, err, "entity conversion failed") {
			return
		}

		uniqueTypes[out.OperationType] = true
		entities = append(entities, entity)
		types = append(types, out.OperationType)
	}

	if errs = db.WithContext(ctx).Transaction(func(tx database.DB) (errs error) {
		if err := tx.UpsertInBatches(entities, l.BatchSize).Error; ctx.Error(&errs, err, "failed to update entities in db") {
			return
		}

		if err := tx.Model(new(InvestOperationType)).
			Where("operation_type not in ?", types).
			Update("deleted", true).
			Error; ctx.Error(&errs, err, "failed to mark deleted entities in db") {
			return
		}

		return
	}); errs != nil {
		return
	}

	ctx.Info("updated entities in db", "count", len(entities))
	return
}
