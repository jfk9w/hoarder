package jobs

import (
	"go.uber.org/multierr"

	"github.com/jfk9w/hoarder/internal/common"
)

type Interface interface {
	ID() string
	Run(ctx Context) error
}

type exclusiveJob struct {
	job   Interface
	users common.MultiMutex[string]
}

func (j *exclusiveJob) ID() string {
	return j.job.ID()
}

func (j *exclusiveJob) Run(ctx Context) (errs error) {
	cancel, err := j.users.TryLock(ctx.User())
	if ctx.Error(&errs, err, "already running") {
		return
	}

	defer cancel()
	return j.job.Run(ctx)
}

type Registry struct {
	jobs []exclusiveJob
}

type jobFilterFunc func(id string) bool

func (r *Registry) Register(job Interface) {
	r.jobs = append(r.jobs, exclusiveJob{job: job})
}

func (r *Registry) Run(ctx Context, jobIDs []string) (errs error) {
	var filter jobFilterFunc
	if len(jobIDs) == 0 || jobIDs[0] == "all" {
		filter = func(_ string) bool { return true }
	} else {
		uniqueJobIDs := make(map[string]bool)
		for _, jobID := range jobIDs {
			uniqueJobIDs[jobID] = true
		}

		filter = func(id string) bool { return uniqueJobIDs[id] }
	}

	for i := range r.jobs {
		job := &r.jobs[i]
		id := job.ID()
		if !filter(id) {
			continue
		}

		ctx := ctx.With("job", id)
		err := job.Run(ctx)
		_ = multierr.AppendInto(&errs, err)
	}

	return
}
