package tinkoff

import (
	"context"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/tinkoff-api"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/etl"
)

type sessionStorage struct {
	db based.Ref[*gorm.DB]
}

func (s *sessionStorage) LoadSession(ctx context.Context, phone string) (*tinkoff.Session, error) {
	db, err := s.db(ctx)
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

	return etl.ToViaJSON[*tinkoff.Session](entity)
}

func (s *sessionStorage) UpdateSession(ctx context.Context, phone string, session *tinkoff.Session) error {
	db, err := s.db(ctx)
	if err != nil {
		return errors.Wrap(err, "get db handle")
	}

	db = db.WithContext(ctx)

	if session == nil {
		return errors.Wrap(db.Delete(new(Session), phone).Error, "delete tokens from db")
	}

	entity, err := etl.ToViaJSON[*Session](session)
	if err != nil {
		return err
	}

	entity.UserPhone = phone

	if err := db.Clauses(etl.Upsert("user_phone")).Create(entity).Error; err != nil {
		return errors.Wrap(err, "upsert session record")
	}

	return nil
}
