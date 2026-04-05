package mqtt

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

// Client implements an MQTT-style microservice client using TCP.
// Messages are routed by topic (the Pattern.Cmd field).
type Client struct {
	opts    Options
	conn    net.Conn
	scanner *bufio.Scanner
	mu      sync.Mutex
	pending map[string]chan mqttResponse
	pendMu  sync.Mutex
	nextID  atomic.Int64
	done    chan struct{}
}

// NewClient creates a new MQTT-style microservice client.
func NewClient(opts Options) *Client {
	return &Client{
		opts:    opts,
		pending: make(map[string]chan mqttResponse),
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

	req := mqttRequest{
		Pattern: pattern,
		ID:      id,
		Topic:   pattern.Cmd,
		QoS:     c.opts.QoS,
	}

	if err := c.populateRequest(&req, data); err != nil {
		return nil, err
	}

	respCh := make(chan mqttResponse, 1)
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
	req := mqttRequest{
		Pattern: pattern,
		IsEvent: true,
		Topic:   pattern.Cmd,
		QoS:     c.opts.QoS,
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
		var resp mqttResponse
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

// populateRequest fills in the request fields from either plain data or an MqttRecord.
func (c *Client) populateRequest(req *mqttRequest, data any) error {
	if rec, ok := data.(*MqttRecord); ok {
		d, err := json.Marshal(rec.Data)
		if err != nil {
			return err
		}
		req.Data = d
		req.Headers = rec.Headers
		if rec.Topic != "" {
			req.Topic = rec.Topic
		}
		req.QoS = rec.QoS
		req.Retained = rec.Retained
	} else {
		d, err := json.Marshal(data)
		if err != nil {
			return err
		}
		req.Data = d
	}
	return nil
}
