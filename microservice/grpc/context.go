package grpc

import "github.com/0xfurai/gonest/microservice"

// GrpcContext carries gRPC-specific metadata for a received message.
type GrpcContext struct {
	*microservice.MessageContext
	serviceName string
	methodName  string
}

// NewGrpcContext creates a new GrpcContext.
func NewGrpcContext(ctx *microservice.MessageContext, serviceName, methodName string) *GrpcContext {
	return &GrpcContext{
		MessageContext: ctx,
		serviceName:    serviceName,
		methodName:     methodName,
	}
}

// ServiceName returns the gRPC service name.
func (c *GrpcContext) ServiceName() string { return c.serviceName }

// MethodName returns the gRPC method name.
func (c *GrpcContext) MethodName() string { return c.methodName }
