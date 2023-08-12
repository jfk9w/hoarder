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

	cancel := based.GoWithFeedback(ctx, context.WithCancel, func(ctx context.Context) {
		for {
			fmt.Printf("Enter username: ")
			username, ok := scan()
			if !ok {
				return
			}

			log := b.Log.With(slog.String("username", username))
			ctx := etl.WithRequestInputFunc(ctx, requestInput)
			reply := "✔ OK"
			if err := b.Pipelines.Run(ctx, log, username); err != nil {
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
	})

	return cancel, nil
}

func requestInput(ctx context.Context, text string) (string, error) {
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
