package main

import (
	"context"
	"fmt"
	"sync"
)

// JobRunner consumes job IDs from RabbitMQ and processes them concurrently.
type JobRunner struct {
	store   *Store
	mq      *MQ
	workers int
	wg      sync.WaitGroup
	cancel  context.CancelFunc
}

func NewJobRunner(s *Store, mq *MQ, workers int) *JobRunner {
	if workers < 1 {
		workers = 4
	}
	return &JobRunner{
		store:   s,
		mq:      mq,
		workers: workers,
	}
}

func (r *JobRunner) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	r.cancel = cancel

	msgs, err := r.mq.Consume()
	if err != nil {
		logError("failed to start consuming from rabbitmq", withErr(err))
		return
	}

	for i := 0; i < r.workers; i++ {
		r.wg.Add(1)
		go func(id int) {
			defer r.wg.Done()
			for {
				select {
				case <-ctx.Done():
					return
				case msg, ok := <-msgs:
					if !ok {
						return
					}
					jobID := string(msg.Body)
					logInfo(fmt.Sprintf("runner[%d] processing job %s", id, jobID))
					r.processJob(ctx, jobID)
					msg.Ack(false) // Acknowledge after processing
				}
			}
		}(i)
	}
	logInfo(fmt.Sprintf("job runner started: %d consumers on rabbitmq queue", r.workers))
}

func (r *JobRunner) Stop() {
	if r.cancel != nil {
		r.cancel()
	}
	r.wg.Wait()
	logInfo("job runner stopped")
}

// Enqueue publishes a job ID to RabbitMQ.
func (r *JobRunner) Enqueue(jobID string) {
	if err := r.mq.Publish(jobID); err != nil {
		logError("failed to enqueue job to rabbitmq: "+jobID, withErr(err))
	}
}

func (r *JobRunner) processJob(parentCtx context.Context, jobID string) {
	r.store.mu.RLock()
	job, ok := r.store.jobs[jobID]
	r.store.mu.RUnlock()
	if !ok {
		logError("job not found: " + jobID)
		return
	}
	r.store.ExecuteJob(parentCtx, job)
}
