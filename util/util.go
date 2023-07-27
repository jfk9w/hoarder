package util

import (
	"encoding/json"

	"gorm.io/gorm/clause"
)

func ToViaJSON[T any](source any) (target T, err error) {
	data, err := json.Marshal(source)
	if err != nil {
		return
	}

	err = json.Unmarshal(data, &target)
	return
}

func Upsert(primaryKey string) clause.OnConflict {
	return clause.OnConflict{
		UpdateAll: true,
		Columns:   []clause.Column{{Name: primaryKey}},
	}
}
