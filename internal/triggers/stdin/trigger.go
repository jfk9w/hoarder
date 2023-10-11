package stdin

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/jfk9w-go/based"
	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/jobs"
	"github.com/jfk9w/hoarder/internal/logs"
	"github.com/jfk9w/hoarder/internal/triggers"
)

const TriggerID = "stdin"

type TriggerParams struct {
	Reader io.Reader
	Writer io.Writer
}

type Trigger struct {
	in  io.Reader
	out io.Writer
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
		in:  params.Reader,
		out: params.Writer,
	}, nil
}

func (t *Trigger) ID() string {
	return TriggerID
}

func (t *Trigger) Run(ctx context.Context, log *slog.Logger, job triggers.Jobs) {
	for {
		userID, err := t.ask(ctx, "Enter user: ")
		if err != nil {
			log.Error("failed to get user", logs.Error(err))
			return
		}

		jobIDs, err := t.ask(ctx, "Enter jobs: ")
		if err != nil {
			log.Error("failed to get jobs", logs.Error(err))
			return
		}

		log := log.With(logs.User(userID))
		ctx := jobs.NewContext(ctx, log, userID).WithAskFn(t.ask)

		reply := " ✔ OK"
		if err := job.Run(ctx, strings.Fields(jobIDs)); err != nil {
			var b strings.Builder
			for _, err := range multierr.Errors(err) {
				b.WriteString(" ✘ ")
				b.WriteString(err.Error())
				b.WriteRune('\n')
			}

			reply = strings.Trim(b.String(), "\n")
		}

		if _, err := fmt.Fprintln(t.out, reply); err != nil {
			log.Error("failed to print result", logs.Error(err))
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
