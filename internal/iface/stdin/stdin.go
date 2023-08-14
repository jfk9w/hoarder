package stdin

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"github.com/go-playground/validator/v10"
	"github.com/jfk9w-go/based"
	"github.com/pkg/errors"
	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/etl"
)

type Pipelines interface {
	Run(ctx context.Context, log *slog.Logger, username string) error
}

type Builder struct {
	Pipelines Pipelines    `validate:"required"`
	Log       *slog.Logger `validate:"required"`
}

var validate = &based.Lazy[*validator.Validate]{
	Fn: func(ctx context.Context) (*validator.Validate, error) {
		return validator.New(), nil
	},
}

func (b Builder) Run(ctx context.Context) (context.CancelFunc, error) {
	if validate, err := validate.Get(ctx); err != nil {
		return nil, err
	} else if err := validate.Struct(b); err != nil {
		return nil, err
	}

	h := &handler{
		pipelines: b.Pipelines,
		log:       b.Log,
	}

	return based.GoWithFeedback(ctx, context.WithCancel, h.startLoop), nil
}

type handler struct {
	pipelines Pipelines
	log       *slog.Logger
	mu        based.RWMutex
}

func (h *handler) startLoop(ctx context.Context) {
	for {
		fmt.Printf("Enter username: ")
		username, ok := scan()
		if !ok {
			return
		}

		log := h.log.With(slog.String("username", username))
		ctx := etl.WithRequestInputFunc(ctx, h.requestInput)
		reply := "✔ OK"
		if err := h.pipelines.Run(ctx, log, username); err != nil {
			var b strings.Builder
			for _, err := range multierr.Errors(err) {
				b.WriteString("✘ ")
				b.WriteString(err.Error())
				b.WriteRune('\n')
			}

			reply = strings.Trim(b.String(), "\n")
		}

		fmt.Println(reply)
	}
}

func (h *handler) requestInput(ctx context.Context, text string) (string, error) {
	ctx, cancel := h.mu.Lock(ctx)
	defer cancel()
	if err := ctx.Err(); err != nil {
		return "", err
	}

	fmt.Printf("%s ", text)
	reply, ok := scan()
	if !ok {
		return "", errors.New("interrupted")
	}

	return reply, nil
}

func scan() (string, bool) {
	scanner := bufio.NewScanner(os.Stdin)
	if ok := scanner.Scan(); !ok {
		return "", false
	}

	return scanner.Text(), true
}
