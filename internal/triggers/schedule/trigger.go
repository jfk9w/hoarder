package schedule

import (
	"sync"
	"time"

	"github.com/jfk9w-go/based"

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

func (t *Trigger) Run(ctx triggers.Context, job triggers.Jobs) {
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
			ctx := ctx.As(userID)
			wg.Add(1)
			go func(userID string, jobIDs []string) {
				defer wg.Done()
				_ = job.Run(ctx.Job(), now, userID, jobIDs)
			}(userID, jobIDs)
		}

		wg.Wait()
	}
}
