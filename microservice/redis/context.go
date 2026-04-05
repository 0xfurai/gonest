package redis

import "github.com/0xfurai/gonest/microservice"

// RedisContext carries Redis-specific metadata for a received message.
type RedisContext struct {
	*microservice.MessageContext
	channel string
}

// NewRedisContext creates a new RedisContext.
func NewRedisContext(ctx *microservice.MessageContext, channel string) *RedisContext {
	return &RedisContext{
		MessageContext: ctx,
		channel:        channel,
	}
}

// Channel returns the Redis pub/sub channel name.
func (c *RedisContext) Channel() string { return c.channel }
