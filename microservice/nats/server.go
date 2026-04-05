package nats

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"

	"github.com/gonest/microservice"
)

// natsRequest is the wire format for NATS-style requests.
// The Subject field carries the NATS subject (maps from Pattern.Cmd).
type natsRequest struct {
	Pattern microservice.Pattern `json:"pattern"`
	Data    json.RawMessage      `json:"data"`
	ID      string               `json:"id"`
	IsEvent bool                 `json:"isEvent,omitempty"`
	Subject string               `json:"subject"`
	Headers map[string]string    `json:"headers,omitempty"`
}

// natsResponse is the wire format for NATS-style responses.
type natsResponse struct {
	ID    string          `json:"id"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Server implements a NATS-style microservice server using TCP with
// subject-based routing. Pattern.Cmd is used as the NATS subject.
type Server struct {
	opts            Options
	messageHandlers map[string]microservice.MessageHandler
	eventHandlers   map[string]microservice.EventHandler
	listener        net.Listener
	mu              sync.RWMutex
	done            chan struct{}
}

// NewServer creates a new NATS-style microservice server.
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
		var req natsRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		// Use subject for routing; fall back to pattern cmd
		routeKey := req.Subject
		if routeKey == "" {
			routeKey = req.Pattern.Cmd
		}

		msgCtx := &microservice.MessageContext{
			Pattern:   microservice.Pattern{Cmd: routeKey},
			Transport: microservice.TransportNATS,
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

		var resp natsResponse
		resp.ID = req.ID

		if !ok {
			resp.Error = "no handler for subject: " + routeKey
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
