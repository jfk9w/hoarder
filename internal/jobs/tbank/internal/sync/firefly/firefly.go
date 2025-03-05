package firefly

import (
	"errors"
	"fmt"

	"gorm.io/gorm/schema"

	"github.com/jfk9w/hoarder/internal/database"
	"github.com/jfk9w/hoarder/internal/firefly"
	"github.com/jfk9w/hoarder/internal/jobs"
)

type exception interface {
	GetMessage() firefly.OptString
	GetException() firefly.OptString
}

func exception2error(e exception) error {
	return errors.New(e.GetMessage().
		Or(e.GetException().
			Or(fmt.Sprintf("%T", e))))
}

type Interface interface {
	schema.Tabler
	Sync(ctx jobs.Context, db database.DB, client firefly.Invoker) ([]Interface, error)
}
