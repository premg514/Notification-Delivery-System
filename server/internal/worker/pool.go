package worker

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"notification-system/internal/observability"
	"notification-system/internal/queue"
)

type Processor[T any] interface {
	Process(ctx context.Context, job T) error
}

type Pool[T any] struct {
	name      string
	processor Processor[T]
	jobs      chan queue.Task[T]
	wg        sync.WaitGroup
}

func NewPool[T any](workerCount int, name string, processor Processor[T]) *Pool[T] {
	if workerCount <= 0 {
		workerCount = 1
	}
	if name == "" {
		name = "default"
	}

	return &Pool[T]{
		name:      name,
		processor: processor,
		jobs:      make(chan queue.Task[T], workerCount*4),
	}
}

func (p *Pool[T]) Start(ctx context.Context, workerCount int) {
	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go func(workerID int) {
			defer p.wg.Done()

			for {
				select {
				case <-ctx.Done():
					return
				case task, ok := <-p.jobs:
					if !ok {
						return
					}

					processingCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
					start := time.Now()
					err := p.safeProcess(processingCtx, task.Job, workerID)
					cancel()
					observability.ObserveWorkerJob(p.name, err == nil, time.Since(start))

					if err != nil {
						_ = task.Envelope.Nack(false, true)
						continue
					}

					_ = task.Envelope.Ack(false)
				}
			}
		}(i + 1)
	}
}

func (p *Pool[T]) Submit(ctx context.Context, task queue.Task[T]) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case p.jobs <- task:
		return nil
	}
}

func (p *Pool[T]) Stop() {
	close(p.jobs)
	p.wg.Wait()
}

func (p *Pool[T]) safeProcess(ctx context.Context, job T, workerID int) (err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			slog.Error("panic recovered in worker job",
				"pool", p.name,
				"worker_id", workerID,
				"panic", recovered,
			)
			err = fmt.Errorf("worker panic: %v", recovered)
		}
	}()

	return p.processor.Process(ctx, job)
}
