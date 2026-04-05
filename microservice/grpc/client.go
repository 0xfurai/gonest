package grpc

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"sync"
	"sync/atomic"

	"github.com/0xfurai/gonest/microservice"
)

// Client implements a gRPC-style microservice client using TCP with
// binary length-prefixed framing (4-byte big-endian length + JSON payload).
type Client struct {
	opts    Options
	conn    net.Conn
	mu      sync.Mutex
	pending map[string]chan grpcResponse
	pendMu  sync.Mutex
	nextID  atomic.Int64
	done    chan struct{}
}

// NewClient creates a new gRPC-style microservice client.
func NewClient(opts Options) *Client {
	return &Client{
		opts:    opts,
		pending: make(map[string]chan grpcResponse),
		done:    make(chan struct{}),
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.opts.Address())
	if err != nil {
		return err
	}
	c.conn = conn

	go c.readLoop()
	return nil
}

func (c *Client) Send(ctx context.Context, pattern microservice.Pattern, data any) (json.RawMessage, error) {
	id := fmt.Sprintf("%d", c.nextID.Add(1))

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req := grpcRequest{
		Pattern: pattern,
		Data:    dataBytes,
		ID:      id,
		Service: c.opts.ServiceName,
	}

	respCh := make(chan grpcResponse, 1)
	c.pendMu.Lock()
	c.pending[id] = respCh
	c.pendMu.Unlock()

	defer func() {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
	}()

	reqBytes, _ := json.Marshal(req)

	c.mu.Lock()
	err = writeFrame(c.conn, reqBytes)
	c.mu.Unlock()
	if err != nil {
		return nil, err
	}

	select {
	case resp := <-respCh:
		if resp.Error != "" {
			return nil, fmt.Errorf("remote error: %s", resp.Error)
		}
		return resp.Data, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (c *Client) Emit(ctx context.Context, pattern microservice.Pattern, data any) error {
	dataBytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	req := grpcRequest{
		Pattern: pattern,
		Data:    dataBytes,
		IsEvent: true,
		Service: c.opts.ServiceName,
	}

	reqBytes, _ := json.Marshal(req)

	c.mu.Lock()
	err = writeFrame(c.conn, reqBytes)
	c.mu.Unlock()
	return err
}

func (c *Client) Close() error {
	close(c.done)
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) readLoop() {
	for {
		// Read 4-byte length prefix
		lenBuf := make([]byte, 4)
		if _, err := io.ReadFull(c.conn, lenBuf); err != nil {
			return
		}
		msgLen := binary.BigEndian.Uint32(lenBuf)
		if msgLen == 0 || msgLen > 10*1024*1024 {
			return
		}

		payload := make([]byte, msgLen)
		if _, err := io.ReadFull(c.conn, payload); err != nil {
			return
		}

		var resp grpcResponse
		if err := json.Unmarshal(payload, &resp); err != nil {
			continue
		}

		c.pendMu.Lock()
		ch, ok := c.pending[resp.ID]
		c.pendMu.Unlock()

		if ok {
			ch <- resp
		}
	}
}
