package etl

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type RequestInputFunc func(ctx context.Context, text string) (string, error)

type requestInputFuncKey struct{}

func WithRequestInputFunc(ctx context.Context, fn RequestInputFunc) context.Context {
	return context.WithValue(ctx, requestInputFuncKey{}, fn)
}

func GetRequestInputFunc(ctx context.Context) RequestInputFunc {
	if fn, _ := ctx.Value(requestInputFuncKey{}).(RequestInputFunc); fn != nil {
		return func(ctx context.Context, text string) (string, error) {
			ctx, cancel := context.WithTimeout(ctx, time.Minute)
			defer cancel()
			return fn(ctx, text)
		}
	}

	return nil
}

var RequestInputStdin RequestInputFunc = func(ctx context.Context, text string) (string, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s: ", text)
	text, err := reader.ReadString('\n')
	if err != nil {
		return "", errors.Wrap(err, "read line from stdin")
	}

	return strings.Trim(text, " \n\t\v"), nil
}
