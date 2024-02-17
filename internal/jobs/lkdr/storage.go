package lkdr

import (
	"context"

	"github.com/jfk9w-go/lkdr-api"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/internal/database"
	. "github.com/jfk9w/hoarder/internal/jobs/lkdr/internal/entities"
)

type storage struct {
	db database.DB
}

func (s *storage) LoadTokens(ctx context.Context, phone string) (*lkdr.Tokens, error) {
	var entity Tokens
	if err := s.db.First(&entity, phone).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "get tokens from db")
	}

	return database.ToViaJSON[*lkdr.Tokens](entity)
}

func (s *storage) UpdateTokens(ctx context.Context, phone string, tokens *lkdr.Tokens) error {
	if tokens == nil {
		return errors.Wrap(s.db.Delete(new(Tokens), phone).Error, "delete tokens from db")
	}

	entity, err := database.ToViaJSON[*Tokens](tokens)
	if err != nil {
		return err
	}

	entity.UserPhone = phone

	if err := s.db.Upsert(entity).Error; err != nil {
		return errors.Wrap(err, "save tokens in db")
	}

	return nil
}
