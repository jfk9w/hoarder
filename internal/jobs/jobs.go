package jobs

import (
	"errors"
	"sync"
	"time"

	"github.com/jfk9w/hoarder/internal/common"
)

const All = "all"

var ErrJobUnconfigured = errors.New("job not configured")

type Interface interface {
	Info() Info
	Run(ctx Context, now time.Time, userID string) error
}

type Info struct {
	ID          string
	Description string
}

type Result struct {
	JobID string
	Error error
}

type exclusiveJob struct {
	job   Interface
	users common.MultiMutex[string]
}

func (j *exclusiveJob) Info() Info {
	return j.job.Info()
}

func (j *exclusiveJob) Run(ctx Context, now time.Time, userID string) (errs error) {
	cancel, err := j.users.TryLock(userID)
	if ctx.Error(&errs, err, "already running") {
		return
	}

	defer cancel()
	return j.job.Run(ctx, now, userID)
}

type Registry struct {
	jobs []exclusiveJob
}

func (r *Registry) Register(job Interface) {
	r.jobs = append(r.jobs, exclusiveJob{job: job})
}

func (r *Registry) Info() []Info {
	infos := make([]Info, len(r.jobs)+1)
	for i := range r.jobs {
		infos[i] = r.jobs[i].Info()
	}

	infos[len(r.jobs)] = Info{
		ID:          All,
		Description: "Запуск всех джобов",
	}

	return infos
}

func (r *Registry) Run(ctx Context, now time.Time, userID string, jobIDs []string) []Result {
	var filter func(id string) bool
	if len(jobIDs) == 0 || jobIDs[0] == All {
		filter = func(_ string) bool { return true }
	} else {
		uniqueJobIDs := make(map[string]bool)
		for _, jobID := range jobIDs {
			uniqueJobIDs[jobID] = true
		}

		filter = func(id string) bool { return uniqueJobIDs[id] }
	}

	var (
		results    []Result
		configured = false
		wg         sync.WaitGroup
		mu         sync.Mutex
	)

	for i := range r.jobs {
		job := &r.jobs[i]
		jobID := job.Info().ID
		if !filter(jobID) {
			continue
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			err := job.Run(ctx.withLog("job", jobID), now, userID)
			jobConfigured := !errors.Is(err, ErrJobUnconfigured)

			mu.Lock()
			defer mu.Unlock()
			if !configured && jobConfigured {
				results = nil
				configured = true
			}

			if jobConfigured == configured {
				results = append(results, Result{JobID: jobID, Error: err})
			}
		}()
	}

	wg.Wait()
	return results
}
