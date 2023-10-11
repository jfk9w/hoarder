package schedule

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"github.com/jfk9w-go/based"

	jobs "github.com/jfk9w/hoarder/internal/jobs"
	"github.com/jfk9w/hoarder/internal/logs"
	"github.com/jfk9w/hoarder/internal/triggers"
)

const TriggerID = "schedule"

type Config struct {
	Users    map[string][]string `yaml:"users" doc:"ID пользователей, для которых данные нужно синхронизировать в фоновом режиме."`
	Interval time.Duration       `yaml:"interval,omitempty" default:"30m" doc:"Интервал синхронизации."`
}

type TriggerParams struct {
	Clock  based.Clock `validate:"required"`
	Config *Config     `validate:"required"`
}

type Trigger struct {
	clock    based.Clock
	users    map[string][]string
	interval time.Duration
}

func NewTrigger(params TriggerParams) (*Trigger, error) {
	if err := based.Validate(params); err != nil {
		return nil, err
	}

	return &Trigger{
		clock:    params.Clock,
		users:    params.Config.Users,
		interval: params.Config.Interval,
	}, nil
}

func (t *Trigger) ID() string {
	return TriggerID
}

func (t *Trigger) Run(ctx context.Context, log *slog.Logger, job triggers.Jobs) {
	for {
		now := t.clock.Now()
		timeout := now.Round(t.interval).Sub(now)
		if timeout < 0 {
			timeout += t.interval
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(timeout):
		}

		var wg sync.WaitGroup
		for userID, jobIDs := range t.users {
			wg.Add(1)
			go func(userID string, jobIDs []string) {
				defer wg.Done()
				log := log.With(logs.User(userID))
				ctx := jobs.NewContext(ctx, log, userID)
				_ = job.Run(ctx, jobIDs)
			}(userID, jobIDs)
		}
	}
}
