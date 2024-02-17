package stdin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jfk9w-go/based"
	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/logs"
	"github.com/jfk9w/hoarder/internal/triggers"
)

const TriggerID = "stdin"

type TriggerParams struct {
	Clock  based.Clock `validate:"required"`
	Reader io.Reader
	Writer io.Writer
}

type Trigger struct {
	clock based.Clock
	in    io.Reader
	out   io.Writer
}

func NewTrigger(params TriggerParams) (*Trigger, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	if params.Reader == nil {
		params.Reader = os.Stdin
	}

	if params.Writer == nil {
		params.Writer = os.Stdout
	}

	return &Trigger{
		clock: params.Clock,
		in:    params.Reader,
		out:   params.Writer,
	}, nil
}

func (t *Trigger) ID() string {
	return TriggerID
}

func (t *Trigger) Run(ctx triggers.Context, job triggers.Jobs) {
	for {
		userID, err := t.ask(ctx, "Enter user: ")
		if err != nil {
			ctx.Error("failed to get user", logs.Error(err))
			return
		}

		ctx := ctx.As(userID)
		jobIDs, err := t.ask(ctx, "Enter jobs: ")
		if err != nil {
			ctx.Error("failed to get jobs", logs.Error(err))
			return
		}

		results := job.Run(ctx.Job().WithAskFn(t.ask), t.clock.Now(), userID, strings.Fields(jobIDs))
		var reply strings.Builder
		for _, result := range results {
			if result.Error == nil {
				reply.WriteString(" ✔ ")
				reply.WriteString(result.JobID)
				reply.WriteRune('\n')
			} else {
				for _, err := range multierr.Errors(result.Error) {
					reply.WriteString(" ✘ ")
					reply.WriteString(result.JobID)
					reply.WriteString(": ")
					reply.WriteString(err.Error())
					reply.WriteRune('\n')
				}
			}
		}

		if _, err := fmt.Fprintln(t.out, reply.String()); err != nil {
			ctx.Error("failed to print result", logs.Error(err))
			return
		}
	}
}

func (t *Trigger) ask(_ context.Context, prompt string) (string, error) {
	if _, err := fmt.Fprint(t.out, prompt); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(t.in)
	if ok := scanner.Scan(); !ok {
		return "", errors.New("scan failed")
	}

	return scanner.Text(), nil
}
