package lkdr

import (
	. "github.com/jfk9w/hoarder/internal/jobs/lkdr/internal/entities"
)

var entities = []any{
	new(User),
	new(Tokens),
	new(Brand),
	new(Receipt),
	new(FiscalData),
	new(FiscalDataItem),
}
