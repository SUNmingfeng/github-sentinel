package scheduler

import (
"context"
"log"
"time"
)

type Job interface {
Run(ctx context.Context) error
}

type Scheduler struct {
jobs []Job
}

func (s *Scheduler) Start() {
for _, job := range s.jobs {
go func(j Job) {
for {
err := j.Run(context.Background())
if err != nil {
log.Println("job error:", err)
}
time.Sleep(24 * time.Hour)
}
}(job)
}
}
