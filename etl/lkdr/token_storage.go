package lkdr

import (
	"context"

	"github.com/jfk9w-go/based"
	"github.com/jfk9w-go/lkdr-api"
	"github.com/pkg/errors"
	"gorm.io/gorm"

	"github.com/jfk9w/hoarder/util"
)

type tokenStorage struct {
	db *based.Lazy[*gorm.DB]
}

func (s *tokenStorage) LoadTokens(ctx context.Context, phone string) (*lkdr.Tokens, error) {
	db, err := s.db.Get(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "get db handle")
	}

	db = db.WithContext(ctx)

	var entity Tokens
	if err := db.First(&entity, phone).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, errors.Wrap(err, "get tokens record from db")
	}

	return util.ToViaJSON[*lkdr.Tokens](entity)
}

func (s *tokenStorage) UpdateTokens(ctx context.Context, phone string, tokens *lkdr.Tokens) error {
	db, err := s.db.Get(ctx)
	if err != nil {
		return errors.Wrap(err, "get db handle")
	}

	db = db.WithContext(ctx)

	entity, err := util.ToViaJSON[*Tokens](tokens)
	if err != nil {
		return err
	}

	entity.Phone = phone

	if err := db.Clauses(util.Upsert("phone")).Create(entity).Error; err != nil {
		return errors.Wrap(err, "upsert token record")
	}

	return nil
}
