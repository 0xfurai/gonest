package mqtt

import (
	"bufio"
	"encoding/json"
	"net"
	"sync"

	"github.com/gonest/microservice"
)

// mqttRequest is the wire format for MQTT-style requests.
// The Topic field carries the MQTT topic (maps from Pattern.Cmd).
type mqttRequest struct {
	Pattern  microservice.Pattern `json:"pattern"`
	Data     json.RawMessage      `json:"data"`
	ID       string               `json:"id"`
	IsEvent  bool                 `json:"isEvent,omitempty"`
	Topic    string               `json:"topic"`
	QoS      QoS                  `json:"qos,omitempty"`
	Retained bool                 `json:"retained,omitempty"`
	Headers  map[string]string    `json:"headers,omitempty"`
}

// mqttResponse is the wire format for MQTT-style responses.
type mqttResponse struct {
	ID    string          `json:"id"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Server implements an MQTT-style microservice server using TCP with
// topic-based pub/sub routing. Pattern.Cmd is used as the MQTT topic.
type Server struct {
	opts            Options
	messageHandlers map[string]microservice.MessageHandler
	eventHandlers   map[string]microservice.EventHandler
	listener        net.Listener
	mu              sync.RWMutex
	done            chan struct{}
}

// NewServer creates a new MQTT-style microservice server.
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
		var req mqttRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		// Use topic for routing; fall back to pattern cmd
		routeKey := req.Topic
		if routeKey == "" {
			routeKey = req.Pattern.Cmd
		}

		msgCtx := &microservice.MessageContext{
			Pattern:   microservice.Pattern{Cmd: routeKey},
			Transport: microservice.TransportMQTT,
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

		var resp mqttResponse
		resp.ID = req.ID

		if !ok {
			resp.Error = "no handler for topic: " + routeKey
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
