package schedule

import (
	"sync/atomic"
	"testing"
	"time"
)

func TestScheduler_Interval(t *testing.T) {
	s := NewScheduler()
	var count atomic.Int32

	s.AddInterval("counter", 50*time.Millisecond, func() {
		count.Add(1)
	})

	s.Start()
	time.Sleep(175 * time.Millisecond)
	s.Stop()

	c := count.Load()
	if c < 2 || c > 4 {
		t.Errorf("expected 2-4 invocations, got %d", c)
	}
}

func TestScheduler_Timeout(t *testing.T) {
	s := NewScheduler()
	var called atomic.Int32

	s.AddTimeout("once", 50*time.Millisecond, func() {
		called.Add(1)
	})

	s.Start()
	time.Sleep(150 * time.Millisecond)
	s.Stop()

	if called.Load() != 1 {
		t.Errorf("expected exactly 1 invocation, got %d", called.Load())
	}
}

func TestScheduler_StopJob(t *testing.T) {
	s := NewScheduler()
	var count atomic.Int32

	s.AddInterval("counter", 50*time.Millisecond, func() {
		count.Add(1)
	})

	s.Start()
	time.Sleep(75 * time.Millisecond)
	s.StopJob("counter")
	before := count.Load()
	time.Sleep(100 * time.Millisecond)
	after := count.Load()

	if after != before {
		t.Errorf("expected count to stop at %d, got %d", before, after)
	}
}

func TestScheduler_MultipleJobs(t *testing.T) {
	s := NewScheduler()
	var count1, count2 atomic.Int32

	s.AddInterval("job1", 50*time.Millisecond, func() { count1.Add(1) })
	s.AddInterval("job2", 50*time.Millisecond, func() { count2.Add(1) })

	s.Start()
	time.Sleep(125 * time.Millisecond)
	s.Stop()

	if count1.Load() < 1 || count2.Load() < 1 {
		t.Errorf("expected both jobs to run: job1=%d, job2=%d", count1.Load(), count2.Load())
	}
}

func TestScheduler_Cron(t *testing.T) {
	s := NewScheduler()
	job := s.AddCron("every-minute", "* * * * *", func() {})
	if job.interval != time.Minute {
		t.Errorf("expected 1 minute interval, got %v", job.interval)
	}
}

func TestParseCronInterval(t *testing.T) {
	tests := []struct {
		expr     string
		expected time.Duration
	}{
		{"* * * * *", time.Minute},
		{"*/5 * * * *", 5 * time.Minute},
		{"*/15 * * * *", 15 * time.Minute},
		{"*/30 * * * *", 30 * time.Minute},
		{"0 * * * *", time.Hour},
		{"0 0 * * *", 24 * time.Hour},
	}

	for _, tt := range tests {
		result := parseCronInterval(tt.expr)
		if result != tt.expected {
			t.Errorf("parseCronInterval(%q): expected %v, got %v", tt.expr, tt.expected, result)
		}
	}
}

func TestScheduler_Stop_BeforeStart(t *testing.T) {
	s := NewScheduler()
	s.AddInterval("test", time.Second, func() {})
	// Should not panic
	s.Stop()
}
