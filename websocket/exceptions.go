package websocket

import (
	"encoding/json"
	"fmt"
)

// WsException is the base exception type for WebSocket errors.
// Equivalent to NestJS WsException.
type WsException struct {
	message string
	cause   error
}

func (e *WsException) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
}

func (e *WsException) Cause() error  { return e.cause }
func (e *WsException) Unwrap() error { return e.cause }

func NewWsException(message string) *WsException {
	return &WsException{message: message}
}

func WrapWsException(message string, cause error) *WsException {
	return &WsException{message: message, cause: cause}
}

// WsExceptionFilter catches errors in WebSocket message handlers.
// Equivalent to NestJS BaseWsExceptionFilter.
type WsExceptionFilter interface {
	Catch(err error, client *Client)
}

// WsExceptionFilterFunc is a convenience adapter for function-based WS filters.
type WsExceptionFilterFunc func(err error, client *Client)

func (f WsExceptionFilterFunc) Catch(err error, client *Client) {
	f(err, client)
}

// BaseWsExceptionFilter is the default WebSocket exception filter that handles
// WsException errors and sends error messages back to the client.
type BaseWsExceptionFilter struct{}

func (f *BaseWsExceptionFilter) Catch(err error, client *Client) {
	var message string
	if wsErr, ok := err.(*WsException); ok {
		message = wsErr.Error()
	} else {
		message = err.Error()
	}

	errPayload := map[string]string{"message": message}
	data, _ := json.Marshal(errPayload)
	_ = client.conn.WriteMessage(1, data)
}
