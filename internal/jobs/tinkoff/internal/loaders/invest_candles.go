package loaders

import (
	"time"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/jobs"
	. "github.com/jfk9w/hoarder/internal/jobs/tinkoff/internal/entities"
)

type investCandles struct {
	accountId string
	batchSize int
	overlap   time.Duration
}

func (l investCandles) TableName() string {
	return new(InvestCandle).TableName()
}

func (l investCandles) Load(ctx jobs.Context, client Client, db database.DB) (ll []Interface, errs error) {

}
