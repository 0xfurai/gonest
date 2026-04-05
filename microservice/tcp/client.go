package tcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/0xfurai/gonest/microservice"
)

// Client implements a TCP-based microservice client.
type Client struct {
	opts     microservice.ClientOptions
	conn     net.Conn
	scanner  *bufio.Scanner
	mu       sync.Mutex
	pending  map[string]chan tcpResponse
	pendMu   sync.Mutex
	nextID   atomic.Int64
	done     chan struct{}
}

// NewClient creates a new TCP microservice client.
func NewClient(opts microservice.ClientOptions) *Client {
	return &Client{
		opts:    opts,
		pending: make(map[string]chan tcpResponse),
		done:    make(chan struct{}),
	}
}

func (c *Client) Connect() error {
	conn, err := net.Dial("tcp", c.opts.Address())
	if err != nil {
		return err
	}
	c.conn = conn
	c.scanner = bufio.NewScanner(conn)
	c.scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	go c.readLoop()
	return nil
}

func (c *Client) Send(ctx context.Context, pattern microservice.Pattern, data any) (json.RawMessage, error) {
	id := fmt.Sprintf("%d", c.nextID.Add(1))

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	req := tcpRequest{
		Pattern: pattern,
		Data:    dataBytes,
		ID:      id,
	}

	respCh := make(chan tcpResponse, 1)
	c.pendMu.Lock()
	c.pending[id] = respCh
	c.pendMu.Unlock()

	defer func() {
		c.pendMu.Lock()
		delete(c.pending, id)
		c.pendMu.Unlock()
	}()

	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')

	c.mu.Lock()
	_, err = c.conn.Write(reqBytes)
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

	req := tcpRequest{
		Pattern: pattern,
		Data:    dataBytes,
		IsEvent: true,
	}

	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')

	c.mu.Lock()
	_, err = c.conn.Write(reqBytes)
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
	for c.scanner.Scan() {
		var resp tcpResponse
		if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
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
