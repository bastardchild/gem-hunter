package ai

import (
	"context"
	"log"
	"time"

	"gemhunter/internal/service"
)

// Worker continuously processes incoming quant runs and updates the AI cache.
type Worker struct {
	agent *Agent
	queue chan service.RunResult
	stop  chan struct{}
}

// NewWorker constructs an AI analysis background worker.
func NewWorker(agent *Agent) *Worker {
	return &Worker{
		agent: agent,
		queue: make(chan service.RunResult, 10),
		stop:  make(chan struct{}),
	}
}

// Enqueue submits a run result to be analyzed by AI agents.
func (w *Worker) Enqueue(r service.RunResult) {
	select {
	case w.queue <- r:
	default:
		log.Printf(`{"level":"warn","event":"ai_queue_full","run_id":%q}`, r.RunID)
	}
}

// Start begins processing the queue in the background.
func (w *Worker) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stop:
				return
			case r := <-w.queue:
				t0 := time.Now()
				_, err := w.agent.ProcessRun(ctx, r)
				if err != nil {
					log.Printf(`{"level":"error","event":"ai_process_failed","run_id":%q,"error":%q}`, r.RunID, err.Error())
				} else {
					log.Printf(`{"level":"info","event":"ai_process_complete","run_id":%q,"count":%d,"duration":%q}`, r.RunID, len(r.Stocks), time.Since(t0).String())
				}
			}
		}
	}()
}

// Stop terminates the worker.
func (w *Worker) Stop() {
	close(w.stop)
}
