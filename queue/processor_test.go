package queue

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"
)

func TestQueue_AddAndProcess(t *testing.T) {
	q := NewQueue("test", 10)
	var processed atomic.Int32

	q.Process("email", func(job *Job) error {
		processed.Add(1)
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)

	q.Add("email", map[string]string{"to": "user@example.com"})
	q.Add("email", map[string]string{"to": "admin@example.com"})

	time.Sleep(100 * time.Millisecond)
	q.Close()

	if processed.Load() != 2 {
		t.Errorf("expected 2 processed, got %d", processed.Load())
	}
}

func TestQueue_JobData(t *testing.T) {
	q := NewQueue("test", 10)
	dataCh := make(chan string, 1)

	q.Process("greet", func(job *Job) error {
		var msg string
		json.Unmarshal(job.Data, &msg)
		dataCh <- msg
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)

	q.Add("greet", "hello world")

	select {
	case msg := <-dataCh:
		if msg != "hello world" {
			t.Errorf("expected 'hello world', got %q", msg)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for job")
	}
}

func TestQueue_Retry(t *testing.T) {
	q := NewQueue("test", 10)
	var attempts atomic.Int32

	q.Process("flaky", func(job *Job) error {
		attempts.Add(1)
		if attempts.Load() < 3 {
			return &QueueFullError{Queue: "retry"}
		}
		return nil
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)

	q.Add("flaky", nil, JobOptions{MaxRetries: 5})

	time.Sleep(200 * time.Millisecond)
	q.Close()

	if attempts.Load() < 3 {
		t.Errorf("expected at least 3 attempts, got %d", attempts.Load())
	}
}

func TestQueue_Len(t *testing.T) {
	q := NewQueue("test", 10)
	q.Add("task", nil)
	q.Add("task", nil)

	if q.Len() != 2 {
		t.Errorf("expected 2 pending, got %d", q.Len())
	}
}

func TestQueue_MultipleWorkers(t *testing.T) {
	q := NewQueue("test", 100)
	q.SetWorkers(4)
	var processed atomic.Int32

	q.Process("work", func(job *Job) error {
		time.Sleep(10 * time.Millisecond)
		processed.Add(1)
		return nil
	})

	for i := 0; i < 20; i++ {
		q.Add("work", nil)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)

	time.Sleep(200 * time.Millisecond)
	q.Close()

	if processed.Load() < 15 {
		t.Errorf("expected most jobs processed with 4 workers, got %d", processed.Load())
	}
}

func TestQueue_UnknownProcessor(t *testing.T) {
	q := NewQueue("test", 10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	q.Start(ctx)

	// Adding a job with no registered processor should not panic
	q.Add("unknown", nil)
	time.Sleep(50 * time.Millisecond)
	q.Close()
}

func TestQueue_Full(t *testing.T) {
	q := NewQueue("test", 1)
	q.Add("task", nil)

	_, err := q.Add("task", nil)
	if err == nil {
		t.Fatal("expected queue full error")
	}
}

func TestQueueFullError(t *testing.T) {
	err := &QueueFullError{Queue: "emails"}
	if err.Error() != "queue emails is full" {
		t.Errorf("unexpected error message: %q", err.Error())
	}
}
