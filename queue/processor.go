package queue

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

// Job represents a job in the queue.
type Job struct {
	ID        string
	Name      string
	Data      json.RawMessage
	CreatedAt time.Time
	Attempts  int
	MaxRetries int
}

// ProcessorFunc handles a job.
type ProcessorFunc func(job *Job) error

// Queue manages a named job queue with workers.
type Queue struct {
	name       string
	jobs       chan *Job
	processors map[string]ProcessorFunc
	mu         sync.RWMutex
	workers    int
	done       chan struct{}
	nextID     int
	idMu       sync.Mutex
}

// NewQueue creates a new in-memory queue.
func NewQueue(name string, bufferSize int) *Queue {
	if bufferSize <= 0 {
		bufferSize = 100
	}
	return &Queue{
		name:       name,
		jobs:       make(chan *Job, bufferSize),
		processors: make(map[string]ProcessorFunc),
		workers:    1,
		done:       make(chan struct{}),
	}
}

// Process registers a handler for jobs with the given name.
func (q *Queue) Process(jobName string, fn ProcessorFunc) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.processors[jobName] = fn
}

// Add enqueues a new job.
func (q *Queue) Add(jobName string, data any, opts ...JobOptions) (*Job, error) {
	var opt JobOptions
	if len(opts) > 0 {
		opt = opts[0]
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	q.idMu.Lock()
	q.nextID++
	id := q.nextID
	q.idMu.Unlock()

	job := &Job{
		ID:         itoa(id),
		Name:       jobName,
		Data:       dataBytes,
		CreatedAt:  time.Now(),
		MaxRetries: opt.MaxRetries,
	}

	select {
	case q.jobs <- job:
		return job, nil
	default:
		return nil, &QueueFullError{Queue: q.name}
	}
}

// Start begins processing jobs with the configured number of workers.
func (q *Queue) Start(ctx context.Context) {
	for i := 0; i < q.workers; i++ {
		go q.worker(ctx)
	}
}

// SetWorkers sets the number of concurrent workers.
func (q *Queue) SetWorkers(n int) {
	q.workers = n
}

// Len returns the number of pending jobs.
func (q *Queue) Len() int {
	return len(q.jobs)
}

// Close stops the queue.
func (q *Queue) Close() {
	close(q.done)
}

func (q *Queue) worker(ctx context.Context) {
	for {
		select {
		case job := <-q.jobs:
			q.mu.RLock()
			processor, ok := q.processors[job.Name]
			q.mu.RUnlock()

			if !ok {
				continue
			}

			job.Attempts++
			err := processor(job)
			if err != nil && job.Attempts <= job.MaxRetries {
				// Re-queue for retry
				select {
				case q.jobs <- job:
				default:
				}
			}

		case <-ctx.Done():
			return
		case <-q.done:
			return
		}
	}
}

// JobOptions configures a job.
type JobOptions struct {
	MaxRetries int
	Delay      time.Duration
}

// QueueFullError is returned when the queue buffer is full.
type QueueFullError struct {
	Queue string
}

func (e *QueueFullError) Error() string {
	return "queue " + e.Queue + " is full"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}
