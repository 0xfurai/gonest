package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync/atomic"

	"github.com/0xfurai/gonest"
	"github.com/0xfurai/gonest/queue"
)

// --- Audio Service ---

type AudioService struct {
	queue     *queue.Queue
	processed atomic.Int64
}

func NewAudioService() *AudioService {
	s := &AudioService{}
	s.queue = queue.NewQueue("audio", 100)

	s.queue.Process("transcode", func(job *queue.Job) error {
		var data map[string]string
		json.Unmarshal(job.Data, &data)
		log.Printf("Transcoding file: %s (job %s)", data["file"], job.ID)
		s.processed.Add(1)
		return nil
	})

	s.queue.Start(context.Background())
	return s
}

func (s *AudioService) Transcode(file string) (*queue.Job, error) {
	return s.queue.Add("transcode", map[string]string{"file": file})
}

func (s *AudioService) Pending() int {
	return s.queue.Len()
}

func (s *AudioService) Processed() int64 {
	return s.processed.Load()
}

// --- Controller ---

type AudioController struct {
	service *AudioService
}

func NewAudioController(service *AudioService) *AudioController {
	return &AudioController{service: service}
}

func (c *AudioController) Register(r gonest.Router) {
	r.Prefix("/audio")

	r.Post("/transcode", c.transcode).HttpCode(http.StatusCreated)
	r.Get("/status", c.status)
}

func (c *AudioController) transcode(ctx gonest.Context) error {
	var body struct {
		File string `json:"file"`
	}
	if err := ctx.Bind(&body); err != nil {
		return err
	}
	if body.File == "" {
		body.File = "audio.mp3"
	}
	job, err := c.service.Transcode(body.File)
	if err != nil {
		return gonest.NewInternalServerError(err.Error())
	}
	return ctx.JSON(http.StatusCreated, map[string]string{
		"jobId":  job.ID,
		"status": "queued",
	})
}

func (c *AudioController) status(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, map[string]any{
		"pending":   c.service.Pending(),
		"processed": c.service.Processed(),
	})
}

// --- Module ---

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewAudioController},
	Providers:   []any{NewAudioService},
})

func main() {
	app := gonest.Create(AppModule)
	log.Fatal(app.Listen(":3000"))
}
