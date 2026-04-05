package main

import (
	"fmt"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/gonest"
	"github.com/gonest/schedule"
)

var taskCount atomic.Int64

type TasksService struct {
	scheduler *schedule.Scheduler
}

func NewTasksService() *TasksService {
	s := &TasksService{scheduler: schedule.NewScheduler()}

	// Run every 10 seconds
	s.scheduler.AddInterval("heartbeat", 10*time.Second, func() {
		taskCount.Add(1)
		fmt.Printf("Heartbeat #%d\n", taskCount.Load())
	})

	// Run once after 5 seconds
	s.scheduler.AddTimeout("startup-task", 5*time.Second, func() {
		fmt.Println("Startup task executed!")
	})

	s.scheduler.Start()
	return s
}

type HealthController struct {
	tasks *TasksService
}

func NewHealthController(tasks *TasksService) *HealthController {
	return &HealthController{tasks: tasks}
}

func (c *HealthController) Register(r gonest.Router) {
	// Health check with heartbeat counter
	r.Get("/health", c.health)
}

func (c *HealthController) health(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{
		"status":     "ok",
		"heartbeats": taskCount.Load(),
	})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewHealthController},
	Providers:   []any{NewTasksService},
})

func main() {
	app := gonest.Create(AppModule)
	log.Fatal(app.Listen(":3000"))
}
