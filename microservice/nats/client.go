package nats

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"sync"
	"sync/atomic"

	"github.com/gonest/microservice"
)

// Client implements a NATS-style microservice client using TCP.
// Messages are routed by subject (the Pattern.Cmd field).
type Client struct {
	opts    Options
	conn    net.Conn
	scanner *bufio.Scanner
	mu      sync.Mutex
	pending map[string]chan natsResponse
	pendMu  sync.Mutex
	nextID  atomic.Int64
	done    chan struct{}
}

// NewClient creates a new NATS-style microservice client.
func NewClient(opts Options) *Client {
	return &Client{
		opts:    opts,
		pending: make(map[string]chan natsResponse),
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

	req := natsRequest{
		Pattern: pattern,
		ID:      id,
		Subject: pattern.Cmd,
	}

	if err := c.populateRequest(&req, data); err != nil {
		return nil, err
	}

	respCh := make(chan natsResponse, 1)
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
	_, err := c.conn.Write(reqBytes)
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
	req := natsRequest{
		Pattern: pattern,
		IsEvent: true,
		Subject: pattern.Cmd,
	}

	if err := c.populateRequest(&req, data); err != nil {
		return err
	}

	reqBytes, _ := json.Marshal(req)
	reqBytes = append(reqBytes, '\n')

	c.mu.Lock()
	_, err := c.conn.Write(reqBytes)
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
		var resp natsResponse
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

// populateRequest fills in the request fields from either plain data or a NatsRecord.
func (c *Client) populateRequest(req *natsRequest, data any) error {
	if rec, ok := data.(*NatsRecord); ok {
		d, err := json.Marshal(rec.Data)
		if err != nil {
			return err
		}
		req.Data = d
		req.Headers = rec.Headers
	} else {
		d, err := json.Marshal(data)
		if err != nil {
			return err
		}
		req.Data = d
	}
	return nil
}
