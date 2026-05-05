package plugins

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type rpcHandler func(ctx context.Context, method string, params json.RawMessage) (any, error)

type rpcMessage struct {
	ID              string          `json:"id,omitempty"`
	ProtocolVersion string          `json:"protocolVersion,omitempty"`
	Method          string          `json:"method,omitempty"`
	Params          json.RawMessage `json:"params,omitempty"`
	Result          json.RawMessage `json:"result,omitempty"`
	Error           *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    string          `json:"code,omitempty"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type rpcClient struct {
	pluginID string
	in       io.WriteCloser
	out      io.Reader
	mu       sync.Mutex
	nextID   atomic.Uint64
	pending  map[string]chan rpcMessage
	closed   chan struct{}
	closeMu  sync.Once
	handler  rpcHandler
}

func newRPCClient(pluginID string, in io.WriteCloser, out io.Reader, handler rpcHandler) *rpcClient {
	c := &rpcClient{
		pluginID: pluginID,
		in:       in,
		out:      out,
		pending:  map[string]chan rpcMessage{},
		closed:   make(chan struct{}),
		handler:  handler,
	}
	go c.readLoop()
	return c
}

func (c *rpcClient) readLoop() {
	scanner := bufio.NewScanner(c.out)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		var msg rpcMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			continue
		}
		if msg.Method != "" {
			c.handleRequest(msg)
			continue
		}
		if msg.ID == "" {
			continue
		}
		c.mu.Lock()
		ch := c.pending[msg.ID]
		delete(c.pending, msg.ID)
		c.mu.Unlock()
		if ch != nil {
			ch <- msg
			close(ch)
		}
	}
	c.close()
}

func (c *rpcClient) handleRequest(msg rpcMessage) {
	if c.handler == nil || msg.ID == "" {
		_ = c.writeMessage(rpcMessage{
			ID:              msg.ID,
			ProtocolVersion: "1.0",
			Error:           &rpcError{Code: "METHOD_NOT_SUPPORTED", Message: "host rpc handler is not available"},
		})
		return
	}

	go func() {
		result, err := c.handler(context.Background(), msg.Method, msg.Params)
		response := rpcMessage{
			ID:              msg.ID,
			ProtocolVersion: "1.0",
		}
		if err != nil {
			response.Error = &rpcError{Code: "HOST_RPC_ERROR", Message: err.Error()}
		} else if result != nil {
			raw, marshalErr := json.Marshal(result)
			if marshalErr != nil {
				response.Error = &rpcError{Code: "HOST_RPC_ERROR", Message: marshalErr.Error()}
			} else {
				response.Result = raw
			}
		}
		_ = c.writeMessage(response)
	}()
}

func (c *rpcClient) call(ctx context.Context, method string, params any, timeout time.Duration, out any) error {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	callCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	id := fmt.Sprintf("%s-%d", c.pluginID, c.nextID.Add(1))
	rawParams, err := json.Marshal(params)
	if err != nil {
		return err
	}
	msg := rpcMessage{
		ID:              id,
		ProtocolVersion: "1.0",
		Method:          method,
		Params:          rawParams,
	}
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	ch := make(chan rpcMessage, 1)
	c.mu.Lock()
	c.pending[id] = ch
	c.mu.Unlock()

	c.mu.Lock()
	_, err = c.in.Write(append(data, '\n'))
	c.mu.Unlock()
	if err != nil {
		c.dropPending(id)
		return err
	}

	select {
	case <-callCtx.Done():
		c.dropPending(id)
		return callCtx.Err()
	case <-c.closed:
		c.dropPending(id)
		return fmt.Errorf("plugin process closed")
	case res := <-ch:
		if res.Error != nil {
			return fmt.Errorf("plugin rpc error: %s", res.Error.Message)
		}
		if out != nil && len(res.Result) > 0 {
			if err := json.Unmarshal(res.Result, out); err != nil {
				return err
			}
		}
		return nil
	}
}

func (c *rpcClient) writeMessage(msg rpcMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err = c.in.Write(append(data, '\n'))
	return err
}

func (c *rpcClient) dropPending(id string) {
	c.mu.Lock()
	delete(c.pending, id)
	c.mu.Unlock()
}

func (c *rpcClient) close() {
	c.closeMu.Do(func() {
		close(c.closed)
		c.mu.Lock()
		for id, ch := range c.pending {
			delete(c.pending, id)
			close(ch)
		}
		c.mu.Unlock()
	})
}
