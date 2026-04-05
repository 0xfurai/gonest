package mongo

import (
	"fmt"

	"github.com/0xfurai/gonest"
)

// Options configures the MongoDB module.
type Options struct {
	URI      string
	Database string
	Host     string
	Port     int
}

// ConnectionURI returns the connection string.
func (o Options) ConnectionURI() string {
	if o.URI != "" {
		return o.URI
	}
	host := o.Host
	if host == "" {
		host = "localhost"
	}
	port := o.Port
	if port == 0 {
		port = 27017
	}
	return fmt.Sprintf("mongodb://%s:%d", host, port)
}

// Connection represents a MongoDB connection (stub).
// In a real implementation, this wraps *mongo.Client from go.mongodb.org/mongo-driver.
type Connection struct {
	URI      string
	Database string
}

// NewModule creates a MongoDB module that provides a Connection.
// The application should import the mongo-driver and use this connection info.
func NewModule(opts Options) *gonest.Module {
	conn := &Connection{
		URI:      opts.ConnectionURI(),
		Database: opts.Database,
	}

	return gonest.NewModule(gonest.ModuleOptions{
		Providers: []any{gonest.ProvideValue[*Connection](conn)},
		Exports:   []any{(*Connection)(nil)},
		Global:    true,
	})
}

// Schema defines the structure/validation for a MongoDB collection.
type Schema struct {
	Collection string
	Indexes    []Index
}

// Index defines a MongoDB index.
type Index struct {
	Fields []string
	Unique bool
}
