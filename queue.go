package main

import (
	"log/slog"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
)

const (
	queueShutdownInterval = 1 * time.Second
	workerStealInterval   = 100 * time.Millisecond
)

// Task to be executed by the queue.
type Task func() error

// TaskQueue represents a queue of tasks to be executed.
type TaskQueue struct {
	mu    sync.Mutex
	tasks []Task
}

// NewTaskQueue creates a new TaskQueue.
func NewTaskQueue() *TaskQueue {
	return &TaskQueue{
		tasks: make([]Task, 0),
	}
}

// PushTask adds a task to the queue.
func (q *TaskQueue) PushTask(task Task) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.tasks = append(q.tasks, task)
}

// PopTask retrieves the next task from the queue.
func (q *TaskQueue) PopTask() Task {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.tasks) == 0 {
		return nil
	}
	task := q.tasks[0]
	q.tasks = q.tasks[1:]
	return task
}

// Worker executes tasks from the queue.
type Worker struct {
	id    uuid.UUID
	queue *TaskQueue
	pool  *WorkerPool
	wg    *sync.WaitGroup
}

// NewWorker creates a new Worker.
func NewWorker(id uuid.UUID, queue *TaskQueue, pool *WorkerPool, wg *sync.WaitGroup) *Worker {
	return &Worker{id, queue, pool, wg}
}

// Start starts the worker.
func (w *Worker) Start(startedChan chan bool) {
	go func(ch chan bool) {
		ch <- true
		for {
			task := w.queue.PopTask()
			if task == nil {
				task = w.pool.Steal(w.id)
			}

			if task == nil {
				time.Sleep(workerStealInterval)
				continue
			}

			if err := task(); err != nil {
				slog.Error("error executing task", "worker_id", w.id, "error", err)
				continue
			}

			slog.Info("task finished", "worker_id", w.id)
		}
	}(startedChan)

	slog.Info("worker started", "id", w.id)
}

// Shutdown waits for the worker to finish its current task and then stops it.
func (w *Worker) Shutdown(timeout time.Duration) {
	go func() {
		defer w.wg.Done()

		slog.Info("shutting down worker", "worker_id", w.id, "timeout", timeout)

		timeoutCh := time.After(timeout)
		for len(w.queue.tasks) > 0 {
			select {
			case <-timeoutCh:
				slog.Warn("timeout reached while shutting down task queue", "worker_id", w.id, "remaining_tasks", len(w.queue.tasks))
				return
			default:
				slog.Warn("waiting for task queue to finish", "worker_id", w.id, "remaining_tasks", len(w.queue.tasks))
				time.Sleep(queueShutdownInterval)
			}
		}
	}()
}

// WorkerPool handles multiple workers and enables task stealing.
type WorkerPool struct {
	wg      *sync.WaitGroup
	workers []*Worker
}

// NewWorkerPool creates a new WorkerPool.
func NewWorkerPool(n int) *WorkerPool {
	pool := &WorkerPool{
		wg:      &sync.WaitGroup{},
		workers: make([]*Worker, n),
	}

	for i := range len(pool.workers) {
		pool.workers[i] = NewWorker(uuid.New(), NewTaskQueue(), pool, pool.wg)
	}

	return pool
}

// Steal attempts to steal a task from another worker.
func (p *WorkerPool) Steal(thiefID uuid.UUID) Task {
	for _, w := range p.workers {
		if w.id != thiefID {
			if task := w.queue.PopTask(); task != nil {
				slog.Debug("stealing task", "from_worker_id", w.id, "to_worker_id", thiefID)
				return task
			}
		}
	}
	return nil
}

// SubmitTask adds a task to a random worker in the pool.
func (p *WorkerPool) SubmitTask(task Task) {
	if len(p.workers) == 0 {
		return
	}
	w := p.workers[rand.Intn(len(p.workers))]
	w.queue.PushTask(task)
}

// StartScheduler add tasks to a worker in the pool periodically.
func (p *WorkerPool) StartScheduler(ticker time.Ticker, tasks ...Task) {
	go func() {
		for {
			select {
			case <-ticker.C:
				for _, task := range tasks {
					go p.SubmitTask(task)
				}
			}
		}
	}()
}

// Start starts all workers in the pool.
func (p *WorkerPool) Start() {
	workerStateCh := make(chan bool)

	for _, w := range p.workers {
		w.Start(workerStateCh)
	}

	for range len(p.workers) {
		p.wg.Add(1)
		<-workerStateCh
	}

	for {
		slog.Debug("worker pool is waiting for tasks")
		time.Sleep(workerStealInterval)
	}
}

// Shutdown waits for all workers to finish their current tasks and then stops
// them.
func (p *WorkerPool) Shutdown(timeout time.Duration) {
	for _, w := range p.workers {
		w.Shutdown(timeout)
	}

	// Timeout is already handled by the Shutdown method of each worker, so we
	// simply wait for all workers to finish their tasks.
	p.wg.Wait()
}
