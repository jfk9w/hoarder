package loaders

import (
	"github.com/AlekSi/pointer"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tbank/internal/entities"
)

type ClientOffers struct {
	Phone     string
	BatchSize int
}

func (l ClientOffers) TableName() string {
	return new(ClientOffer).TableName()
}

func (l ClientOffers) Load(ctx jobs.Context, client Client, db database.DB) (ls []Interface, errs error) {
	out, err := client.ClientOfferEssences(ctx)
	if ctx.Error(&errs, err, "failed to get data from api") {
		return
	}

	if len(out) == 0 {
		return
	}

	entities, err := database.ToViaJSON[[]ClientOffer](out)
	if ctx.Error(&errs, err, "entity conversion failed") {
		return
	}

	for i := range out {
		entity := &entities[i]
		entity.UserPhone = l.Phone
		for _, accountId := range out[i].AccountIds {
			entity.Accounts = append(entity.Accounts, ClientOfferAccount{AccountId: accountId})
			for j := range out[i].Essences {
				entity := &entity.Essences[j]
				switch entity.ExternalCode {
				case "CATEGORY":
					entity.SpendingCategoryId = pointer.To(entity.ExternalId)
				case "BRAND":
					entity.BrandId = pointer.To(entity.ExternalId)
				}

				for _, mccCode := range out[i].Essences[j].MccCodes {
					entity.MccCodes = append(entity.MccCodes, ClientOfferEssenceMccCode{
						MccCode: mccCode,
					})
				}
			}
		}
	}

	if err := db.WithContext(ctx).
		UpsertInBatches(entities, l.BatchSize).
		Error; ctx.Error(&errs, err, "failed to update entities in db") {
		return
	}

	ctx.Info("updated entities in db", "count", len(entities))
	return
}
