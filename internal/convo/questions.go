package convo

import (
	"context"

	"github.com/jfk9w-go/based"
	"github.com/pkg/errors"
)

var (
	ErrDuplicateQuestion = errors.New("duplicate question")
	ErrNoQuestions       = errors.New("no questions")
)

type (
	AskFunc[K comparable] func(ctx context.Context, key K) error

	Asker[K comparable, V any] interface {
		Ask(ctx context.Context, key K, fn AskFunc[K]) (V, error)
	}

	Answerer[K comparable, V any] interface {
		Answer(ctx context.Context, key K, answer V) error
	}

	Questions[K comparable, V any] interface {
		Asker[K, V]
		Answerer[K, V]
	}
)

type questions[K comparable, V any] struct {
	questions map[K]chan V
	mu        based.RWMutex
}

func NewQuestions[K comparable, V any]() Questions[K, V] {
	return &questions[K, V]{
		questions: make(map[K]chan V),
	}
}

func (qs *questions[K, V]) createQuestion(ctx context.Context, key K, fn AskFunc[K]) (chan V, error) {
	ctx, cancel := qs.mu.Lock(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	if _, ok := qs.questions[key]; ok {
		return nil, ErrDuplicateQuestion
	}

	if err := fn(ctx, key); err != nil {
		return nil, errors.Wrap(err, "ask")
	}

	question := make(chan V)
	qs.questions[key] = question

	return question, nil
}

func (qs *questions[K, V]) getQuestion(ctx context.Context, key K) (chan V, error) {
	ctx, cancel := qs.mu.Lock(ctx)
	defer cancel()
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	question, ok := qs.questions[key]
	if !ok {
		return nil, ErrNoQuestions
	}

	return question, nil
}

func (qs *questions[K, V]) deleteQuestion(key K) {
	_, cancel := qs.mu.Lock(context.Background())
	defer cancel()
	delete(qs.questions, key)
}

func (qs *questions[K, V]) Ask(ctx context.Context, key K, fn AskFunc[K]) (answer V, err error) {
	question, err := qs.createQuestion(ctx, key, fn)
	if err != nil {
		err = errors.Wrap(err, "create question")
		return
	}

	defer qs.deleteQuestion(key)

	select {
	case answer := <-question:
		return answer, nil
	case <-ctx.Done():
		err = ctx.Err()
		return
	}
}

func (qs *questions[K, V]) Answer(ctx context.Context, key K, answer V) error {
	question, err := qs.getQuestion(ctx, key)
	if err != nil {
		return err
	}

	defer close(question)

	select {
	case question <- answer:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
