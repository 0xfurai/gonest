package rabbitmq

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"

	"github.com/gonest/microservice"
)

// rmqRequest is the wire format for RabbitMQ-style requests.
// The Queue field carries the RabbitMQ queue (maps from Pattern.Cmd).
type rmqRequest struct {
	Pattern    microservice.Pattern `json:"pattern"`
	Data       json.RawMessage      `json:"data"`
	ID         string               `json:"id"`
	IsEvent    bool                 `json:"isEvent,omitempty"`
	Queue      string               `json:"queue"`
	Exchange   string               `json:"exchange,omitempty"`
	RoutingKey string               `json:"routingKey,omitempty"`
	Headers    map[string]string    `json:"headers,omitempty"`
}

// rmqResponse is the wire format for RabbitMQ-style responses.
type rmqResponse struct {
	ID    string          `json:"id"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Server implements a RabbitMQ-style microservice server using TCP with
// queue-based routing. Pattern.Cmd is used as the queue name for routing.
type Server struct {
	opts            Options
	messageHandlers map[string]microservice.MessageHandler
	eventHandlers   map[string]microservice.EventHandler
	listener        net.Listener
	mu              sync.RWMutex
	done            chan struct{}
}

// NewServer creates a new RabbitMQ-style microservice server.
func NewServer(opts Options) *Server {
	return &Server{
		opts:            opts,
		messageHandlers: make(map[string]microservice.MessageHandler),
		eventHandlers:   make(map[string]microservice.EventHandler),
		done:            make(chan struct{}),
	}
}

func (s *Server) AddMessageHandler(pattern microservice.Pattern, handler microservice.MessageHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messageHandlers[pattern.Cmd] = handler
}

func (s *Server) AddEventHandler(pattern microservice.Pattern, handler microservice.EventHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.eventHandlers[pattern.Cmd] = handler
}

func (s *Server) Listen() error {
	ln, err := net.Listen("tcp", s.opts.Address())
	if err != nil {
		return err
	}
	s.listener = ln

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-s.done:
					return
				default:
					continue
				}
			}
			go s.handleConn(conn)
		}
	}()

	return nil
}

func (s *Server) Close() error {
	close(s.done)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

func (s *Server) handleConn(conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		var req rmqRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		// Use queue for routing; fall back to pattern cmd
		routeKey := req.Queue
		if routeKey == "" {
			routeKey = req.Pattern.Cmd
		}

		msgCtx := &microservice.MessageContext{
			Pattern:   microservice.Pattern{Cmd: routeKey},
			Transport: microservice.TransportRabbitMQ,
			Data:      req.Data,
		}

		if req.IsEvent {
			s.mu.RLock()
			handler, ok := s.eventHandlers[routeKey]
			s.mu.RUnlock()
			if ok {
				_ = handler(msgCtx)
			}
			continue
		}

		s.mu.RLock()
		handler, ok := s.messageHandlers[routeKey]
		s.mu.RUnlock()

		var resp rmqResponse
		resp.ID = req.ID

		if !ok {
			resp.Error = "no handler for queue: " + routeKey
		} else {
			result, err := handler(msgCtx)
			if err != nil {
				resp.Error = err.Error()
			} else {
				data, _ := json.Marshal(result)
				resp.Data = data
			}
		}

		respBytes, _ := json.Marshal(resp)
		respBytes = append(respBytes, '\n')
		conn.Write(respBytes)
	}
}
