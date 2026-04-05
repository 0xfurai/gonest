package grpc

import (
	"encoding/binary"
	"encoding/json"
	"io"
	"net"
	"sync"

	"github.com/0xfurai/gonest/microservice"
)

// grpcRequest is the wire format for gRPC-style requests.
type grpcRequest struct {
	Pattern microservice.Pattern `json:"pattern"`
	Data    json.RawMessage      `json:"data"`
	ID      string               `json:"id"`
	IsEvent bool                 `json:"isEvent,omitempty"`
	Service string               `json:"service,omitempty"`
}

// grpcResponse is the wire format for gRPC-style responses.
type grpcResponse struct {
	ID    string          `json:"id"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error string          `json:"error,omitempty"`
}

// Server implements a gRPC-style microservice server using TCP with
// binary length-prefixed framing (4-byte big-endian length + JSON payload).
type Server struct {
	opts            Options
	messageHandlers map[string]microservice.MessageHandler
	eventHandlers   map[string]microservice.EventHandler
	listener        net.Listener
	mu              sync.RWMutex
	done            chan struct{}
}

// NewServer creates a new gRPC-style microservice server.
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

	for {
		// Read 4-byte length prefix (big-endian)
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(conn, lenBuf); err != nil {
			return
		}
		msgLen := binary.BigEndian.Uint32(lenBuf)
		if msgLen == 0 || msgLen > 10*1024*1024 {
			return
		}

		// Read the JSON payload
		payload := make([]byte, msgLen)
		if _, err := io.ReadFull(conn, payload); err != nil {
			return
		}

		var req grpcRequest
		if err := json.Unmarshal(payload, &req); err != nil {
			continue
		}

		msgCtx := &microservice.MessageContext{
			Pattern:   req.Pattern,
			Transport: microservice.TransportGRPC,
			Data:      req.Data,
		}

		if req.IsEvent {
			s.mu.RLock()
			handler, ok := s.eventHandlers[req.Pattern.Cmd]
			s.mu.RUnlock()
			if ok {
				_ = handler(msgCtx)
			}
			continue
		}

		s.mu.RLock()
		handler, ok := s.messageHandlers[req.Pattern.Cmd]
		s.mu.RUnlock()

		var resp grpcResponse
		resp.ID = req.ID

		if !ok {
			resp.Error = "no handler for pattern: " + req.Pattern.Cmd
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
		writeFrame(conn, respBytes)
	}
}

// writeFrame writes a length-prefixed frame to the connection.
func writeFrame(conn net.Conn, data []byte) error {
	lenBuf := make([]byte, 4)
	binary.BigEndian.PutUint32(lenBuf, uint32(len(data)))
	if _, err := conn.Write(lenBuf); err != nil {
		return err
	}
	_, err := conn.Write(data)
	return err
}
