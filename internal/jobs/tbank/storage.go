package tbank

import (
	"context"

	"github.com/jfk9w/hoarder/internal/database"

	tbank "github.com/jfk9w-go/tbank-api"

	"github.com/pkg/errors"
	"gorm.io/gorm"

	. "github.com/jfk9w/hoarder/internal/jobs/tbank/internal/entities"
)

type storage struct {
	db database.DB
}

func (s *storage) LoadSession(ctx context.Context, phone string) (*tbank.Session, error) {
	var entity Session
	if err := s.db.First(&entity, phone).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "get session from db")
	}

	return database.ToViaJSON[*tbank.Session](entity)
}

func (s *storage) UpdateSession(ctx context.Context, phone string, session *tbank.Session) error {
	if session == nil {
		return errors.Wrap(s.db.Delete(new(Session), phone).Error, "delete session from db")
	}

	entity, err := database.ToViaJSON[*Session](session)
	if err != nil {
		return err
	}

	entity.UserPhone = phone

	if err := s.db.Upsert(entity).Error; err != nil {
		return errors.Wrap(err, "save session in db")
	}

	return nil
}
