package schedule

import (
	"sync"
	"time"
)

// Job represents a scheduled job.
type Job struct {
	Name     string
	fn       func()
	interval time.Duration
	cronExpr string
	once     bool
	stop     chan struct{}
	running  bool
}

// Scheduler manages scheduled jobs (cron, interval, timeout).
type Scheduler struct {
	mu   sync.Mutex
	jobs []*Job
}

// NewScheduler creates a new scheduler.
func NewScheduler() *Scheduler {
	return &Scheduler{}
}

// AddInterval schedules a function to run at a fixed interval.
func (s *Scheduler) AddInterval(name string, interval time.Duration, fn func()) *Job {
	job := &Job{
		Name:     name,
		fn:       fn,
		interval: interval,
		stop:     make(chan struct{}),
	}
	s.mu.Lock()
	s.jobs = append(s.jobs, job)
	s.mu.Unlock()
	return job
}

// AddTimeout schedules a function to run once after a delay.
func (s *Scheduler) AddTimeout(name string, delay time.Duration, fn func()) *Job {
	job := &Job{
		Name:     name,
		fn:       fn,
		interval: delay,
		once:     true,
		stop:     make(chan struct{}),
	}
	s.mu.Lock()
	s.jobs = append(s.jobs, job)
	s.mu.Unlock()
	return job
}

// AddCron schedules a function using a simplified cron expression.
// Supports: "* * * * *" (minute hour day month weekday)
// For simplicity, this implementation uses interval-based approximation.
// A production implementation would parse full cron expressions.
func (s *Scheduler) AddCron(name string, expr string, fn func()) *Job {
	interval := parseCronInterval(expr)
	job := &Job{
		Name:     name,
		fn:       fn,
		interval: interval,
		cronExpr: expr,
		stop:     make(chan struct{}),
	}
	s.mu.Lock()
	s.jobs = append(s.jobs, job)
	s.mu.Unlock()
	return job
}

// Start begins executing all registered jobs.
func (s *Scheduler) Start() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range s.jobs {
		if !job.running {
			job.running = true
			go s.runJob(job)
		}
	}
}

// Stop stops all running jobs.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range s.jobs {
		if job.running {
			close(job.stop)
			job.running = false
		}
	}
}

// StopJob stops a specific job by name.
func (s *Scheduler) StopJob(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, job := range s.jobs {
		if job.Name == name && job.running {
			close(job.stop)
			job.running = false
			break
		}
	}
}

func (s *Scheduler) runJob(job *Job) {
	if job.once {
		select {
		case <-time.After(job.interval):
			job.fn()
		case <-job.stop:
		}
		return
	}

	ticker := time.NewTicker(job.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			job.fn()
		case <-job.stop:
			return
		}
	}
}

// parseCronInterval provides a simple cron-to-interval mapping.
// Full cron parsing is complex; this covers common patterns.
func parseCronInterval(expr string) time.Duration {
	switch expr {
	case "* * * * *":
		return time.Minute
	case "*/5 * * * *":
		return 5 * time.Minute
	case "*/15 * * * *":
		return 15 * time.Minute
	case "*/30 * * * *":
		return 30 * time.Minute
	case "0 * * * *":
		return time.Hour
	case "0 0 * * *":
		return 24 * time.Hour
	default:
		return time.Minute
	}
}
