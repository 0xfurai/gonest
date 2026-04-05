package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gonest"
	"github.com/gonest/microservice"
	"github.com/gonest/microservice/tcp"
)

// --- Microservice Server Side ---

func startMicroservice() {
	server := tcp.NewServer(microservice.ServerOptions{
		Host: "127.0.0.1",
		Port: 4000,
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "sum"}, func(ctx *microservice.MessageContext) (any, error) {
		var nums []int
		json.Unmarshal(ctx.Data, &nums)
		sum := 0
		for _, n := range nums {
			sum += n
		}
		return sum, nil
	})

	server.AddMessageHandler(microservice.Pattern{Cmd: "hello"}, func(ctx *microservice.MessageContext) (any, error) {
		var name string
		json.Unmarshal(ctx.Data, &name)
		return fmt.Sprintf("Hello, %s!", name), nil
	})

	if err := server.Listen(); err != nil {
		log.Fatal(err)
	}
	fmt.Println("Microservice listening on :4000")
}

// --- HTTP Gateway Side ---

type MathController struct {
	client *tcp.Client
}

func NewMathController() *MathController {
	client := tcp.NewClient(microservice.ClientOptions{Host: "127.0.0.1", Port: 4000})
	if err := client.Connect(); err != nil {
		log.Fatal(err)
	}
	return &MathController{client: client}
}

func (c *MathController) Register(r gonest.Router) {
	r.Prefix("/math")

	// Compute sum via TCP microservice
	r.Get("/sum", c.sum)

	// Greet by name via TCP microservice
	r.Get("/hello/:name", c.hello)
}

func (c *MathController) sum(ctx gonest.Context) error {
	rawCtx, cancel := context.WithTimeout(ctx.Ctx(), 5*time.Second)
	defer cancel()

	resp, err := c.client.Send(rawCtx, microservice.Pattern{Cmd: "sum"}, []int{1, 2, 3, 4, 5})
	if err != nil {
		return gonest.NewInternalServerError(err.Error())
	}
	var result int
	json.Unmarshal(resp, &result)
	return ctx.JSON(http.StatusOK, map[string]int{"sum": result})
}

func (c *MathController) hello(ctx gonest.Context) error {
	name := ctx.Param("name").(string)
	rawCtx, cancel := context.WithTimeout(ctx.Ctx(), 5*time.Second)
	defer cancel()

	resp, err := c.client.Send(rawCtx, microservice.Pattern{Cmd: "hello"}, name)
	if err != nil {
		return gonest.NewInternalServerError(err.Error())
	}
	var greeting string
	json.Unmarshal(resp, &greeting)
	return ctx.JSON(http.StatusOK, map[string]string{"greeting": greeting})
}

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewMathController},
})

func main() {
	// Start microservice in background
	go startMicroservice()
	time.Sleep(100 * time.Millisecond)

	// Start HTTP gateway
	app := gonest.Create(AppModule)
	log.Fatal(app.Listen(":3000"))
}
