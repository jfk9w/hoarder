package jobs

import (
	"sync"
	"time"

	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/common"
)

type Interface interface {
	ID() string
	Run(ctx Context, now time.Time, userID string) error
}

type Result struct {
	JobID string
	Error error
}

type exclusiveJob struct {
	job   Interface
	users common.MultiMutex[string]
}

func (j *exclusiveJob) ID() string {
	return j.job.ID()
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

func (r *Registry) Run(ctx Context, now time.Time, userID string, jobIDs []string) []Result {
	var filter func(id string) bool
	if len(jobIDs) == 0 || jobIDs[0] == "all" {
		filter = func(_ string) bool { return true }
	} else {
		uniqueJobIDs := make(map[string]bool)
		for _, jobID := range jobIDs {
			uniqueJobIDs[jobID] = true
		}

		filter = func(id string) bool { return uniqueJobIDs[id] }
	}

	var (
		results []Result
		wg      sync.WaitGroup
	)

	for i := range r.jobs {
		job := &r.jobs[i]
		jobID := job.ID()
		if !filter(jobID) {
			continue
		}

		results = append(results, Result{JobID: jobID})
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			err := job.Run(ctx.withLog("job", jobID), now, userID)
			_ = multierr.AppendInto(&results[idx].Error, err)
		}(len(results) - 1)
	}

	wg.Wait()
	return results
}
