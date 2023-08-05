package tinkoff

import (
	"context"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"

	"github.com/jfk9w/hoarder/internal/util"

	"github.com/pkg/errors"
	"gorm.io/gorm"
)

type sessionStorage struct {
	db *based.Lazy[*gorm.DB]
}

func (s *sessionStorage) LoadSession(ctx context.Context, phone string) (*tinkoff.Session, error) {
	db, err := s.db.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get db handle")
	}

	db = db.WithContext(ctx)

	var entity Session
	if err := db.First(&entity, phone).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "get session record from db")
	}

	return util.ToViaJSON[*tinkoff.Session](entity)
}

func (s *sessionStorage) UpdateSession(ctx context.Context, phone string, session *tinkoff.Session) error {
	db, err := s.db.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "get db handle")
	}

	db = db.WithContext(ctx)

	if session == nil {
		return errors.Wrap(db.Delete(new(Session), phone).Error, "delete tokens from db")
	}

	entity, err := util.ToViaJSON[*Session](session)
	if err != nil {
		return err
	}

	entity.UserPhone = phone

	if err := db.Clauses(util.Upsert("user_phone")).Create(entity).Error; err != nil {
		return errors.Wrap(err, "upsert session record")
	}

	return nil
}
