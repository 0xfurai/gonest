package microservice

import "fmt"

// RpcException is the base exception type for microservice RPC errors.
// Equivalent to NestJS RpcException.
type RpcException struct {
	message string
	cause   error
}

func (e *RpcException) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("%s: %s", e.message, e.cause.Error())
	}
	return e.message
}

func (e *RpcException) Cause() error  { return e.cause }
func (e *RpcException) Unwrap() error { return e.cause }

func NewRpcException(message string) *RpcException {
	return &RpcException{message: message}
}

func WrapRpcException(message string, cause error) *RpcException {
	return &RpcException{message: message, cause: cause}
}

// KafkaRetriableException indicates a Kafka message processing failure
// that should be retried.
type KafkaRetriableException struct {
	RpcException
}

func NewKafkaRetriableException(message string) *KafkaRetriableException {
	return &KafkaRetriableException{RpcException: RpcException{message: message}}
}

func IsKafkaRetriable(err error) bool {
	_, ok := err.(*KafkaRetriableException)
	return ok
}

// RpcExceptionFilter catches errors in microservice message handlers.
// Equivalent to NestJS BaseRpcExceptionFilter.
type RpcExceptionFilter interface {
	Catch(err error, ctx *MessageContext) error
}

// RpcExceptionFilterFunc is a convenience adapter for function-based RPC filters.
type RpcExceptionFilterFunc func(err error, ctx *MessageContext) error

func (f RpcExceptionFilterFunc) Catch(err error, ctx *MessageContext) error {
	return f(err, ctx)
}

// BaseRpcExceptionFilter is the default RPC exception filter that handles
// RpcException errors and produces a standard error response.
type BaseRpcExceptionFilter struct{}

func (f *BaseRpcExceptionFilter) Catch(err error, ctx *MessageContext) error {
	if rpcErr, ok := err.(*RpcException); ok {
		return rpcErr
	}
	return NewRpcException(err.Error())
}
