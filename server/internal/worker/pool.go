package worker

import (
	"context"
	"sync"
	"time"

	"notification-system/internal/queue"
)

type Processor[T any] interface {
	Process(ctx context.Context, job T) error
}

type Pool[T any] struct {
	processor Processor[T]
	jobs      chan queue.Task[T]
	wg        sync.WaitGroup
}

func NewPool[T any](workerCount int, processor Processor[T]) *Pool[T] {
	if workerCount <= 0 {
		workerCount = 1
	}

	return &Pool[T]{
		processor: processor,
		jobs:      make(chan queue.Task[T], workerCount*4),
	}
}

func (p *Pool[T]) Start(ctx context.Context, workerCount int) {
	for i := 0; i < workerCount; i++ {
		p.wg.Add(1)
		go func() {
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
					err := p.processor.Process(processingCtx, task.Job)
					cancel()

					if err != nil {
						_ = task.Envelope.Nack(false, true)
						continue
					}

					_ = task.Envelope.Ack(false)
				}
			}
		}()
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
