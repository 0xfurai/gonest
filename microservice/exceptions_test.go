package microservice

import (
	"errors"
	"testing"
)

func TestRpcException(t *testing.T) {
	ex := NewRpcException("rpc error")
	if ex.Error() != "rpc error" {
		t.Errorf("expected 'rpc error', got %q", ex.Error())
	}
	if ex.Cause() != nil {
		t.Error("expected nil cause")
	}
}

func TestRpcException_Wrapped(t *testing.T) {
	cause := errors.New("root cause")
	ex := WrapRpcException("wrapped", cause)
	if ex.Error() != "wrapped: root cause" {
		t.Errorf("expected 'wrapped: root cause', got %q", ex.Error())
	}
	if !errors.Is(ex, cause) {
		t.Error("expected errors.Is to find cause")
	}
}

func TestKafkaRetriableException(t *testing.T) {
	ex := NewKafkaRetriableException("retry me")
	if !IsKafkaRetriable(ex) {
		t.Error("expected IsKafkaRetriable to return true")
	}
	if IsKafkaRetriable(errors.New("other")) {
		t.Error("expected IsKafkaRetriable to return false for other errors")
	}
}

func TestBaseRpcExceptionFilter(t *testing.T) {
	filter := &BaseRpcExceptionFilter{}

	rpcErr := NewRpcException("rpc fail")
	result := filter.Catch(rpcErr, &MessageContext{})
	if result != rpcErr {
		t.Error("expected filter to return the original RpcException")
	}

	genericErr := errors.New("generic")
	result = filter.Catch(genericErr, &MessageContext{})
	if _, ok := result.(*RpcException); !ok {
		t.Error("expected filter to wrap generic error in RpcException")
	}
}
