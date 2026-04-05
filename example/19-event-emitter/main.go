package main

import (
	"log"
	"net/http"
	"sync"
	"sync/atomic"

	"github.com/gonest"
)

// --- Events ---

type OrderCreatedEvent struct {
	OrderID     int    `json:"orderId"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// --- DTOs ---

type CreateOrderDto struct {
	Name        string `json:"name" validate:"required"`
	Description string `json:"description"`
}

type Order struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// --- Event Listener ---

type OrderCreatedListener struct {
	mu  sync.RWMutex
	log []OrderCreatedEvent
}

func NewOrderCreatedListener(emitter *gonest.EventEmitter) *OrderCreatedListener {
	l := &OrderCreatedListener{}
	emitter.On("order.created", func(data any) error {
		event := data.(OrderCreatedEvent)
		l.mu.Lock()
		l.log = append(l.log, event)
		l.mu.Unlock()
		log.Printf("Order created: #%d %s", event.OrderID, event.Name)
		return nil
	})
	return l
}

func (l *OrderCreatedListener) GetLog() []OrderCreatedEvent {
	l.mu.RLock()
	defer l.mu.RUnlock()
	result := make([]OrderCreatedEvent, len(l.log))
	copy(result, l.log)
	return result
}

// --- Orders Service ---

type OrdersService struct {
	mu      sync.RWMutex
	orders  []Order
	nextID  atomic.Int64
	emitter *gonest.EventEmitter
}

func NewOrdersService(emitter *gonest.EventEmitter) *OrdersService {
	return &OrdersService{emitter: emitter}
}

func (s *OrdersService) Create(dto CreateOrderDto) Order {
	id := int(s.nextID.Add(1))
	order := Order{
		ID:          id,
		Name:        dto.Name,
		Description: dto.Description,
	}
	s.mu.Lock()
	s.orders = append(s.orders, order)
	s.mu.Unlock()

	// Emit event
	s.emitter.Emit("order.created", OrderCreatedEvent{
		OrderID:     order.ID,
		Name:        order.Name,
		Description: order.Description,
	})

	return order
}

func (s *OrdersService) FindAll() []Order {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Order, len(s.orders))
	copy(result, s.orders)
	return result
}

// --- Controllers ---

type OrdersController struct {
	service *OrdersService
}

func NewOrdersController(service *OrdersService) *OrdersController {
	return &OrdersController{service: service}
}

func (c *OrdersController) Register(r gonest.Router) {
	r.Prefix("/orders")

	r.Post("/", c.create).HttpCode(http.StatusCreated)
	r.Get("/", c.findAll)
}

func (c *OrdersController) create(ctx gonest.Context) error {
	var dto CreateOrderDto
	if err := ctx.Bind(&dto); err != nil {
		return err
	}
	order := c.service.Create(dto)
	return ctx.JSON(http.StatusCreated, order)
}

func (c *OrdersController) findAll(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.service.FindAll())
}

type EventsController struct {
	listener *OrderCreatedListener
}

func NewEventsController(listener *OrderCreatedListener) *EventsController {
	return &EventsController{listener: listener}
}

func (c *EventsController) Register(r gonest.Router) {
	r.Prefix("/events")

	r.Get("/log", c.getLog)
}

func (c *EventsController) getLog(ctx gonest.Context) error {
	return ctx.JSON(http.StatusOK, c.listener.GetLog())
}

// --- Module ---

var AppModule = gonest.NewModule(gonest.ModuleOptions{
	Controllers: []any{NewOrdersController, NewEventsController},
	Providers:   []any{gonest.NewEventEmitter, NewOrdersService, NewOrderCreatedListener},
})

func main() {
	app := gonest.Create(AppModule)
	app.EnableCors()
	log.Fatal(app.Listen(":3000"))
}
