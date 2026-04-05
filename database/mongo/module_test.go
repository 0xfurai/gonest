package mongo

import "testing"

func TestOptions_ConnectionURI(t *testing.T) {
	opts := Options{URI: "mongodb://admin:pass@db.example.com:27017"}
	if opts.ConnectionURI() != "mongodb://admin:pass@db.example.com:27017" {
		t.Errorf("unexpected URI: %q", opts.ConnectionURI())
	}
}

func TestOptions_ConnectionURI_Default(t *testing.T) {
	opts := Options{}
	if opts.ConnectionURI() != "mongodb://localhost:27017" {
		t.Errorf("unexpected URI: %q", opts.ConnectionURI())
	}
}

func TestOptions_ConnectionURI_CustomHost(t *testing.T) {
	opts := Options{Host: "mongo.local", Port: 27018}
	expected := "mongodb://mongo.local:27018"
	if opts.ConnectionURI() != expected {
		t.Errorf("expected %q, got %q", expected, opts.ConnectionURI())
	}
}

func TestConnection(t *testing.T) {
	conn := &Connection{URI: "mongodb://localhost:27017", Database: "testdb"}
	if conn.URI == "" || conn.Database == "" {
		t.Error("expected non-empty connection fields")
	}
}

func TestSchema(t *testing.T) {
	schema := Schema{
		Collection: "users",
		Indexes: []Index{
			{Fields: []string{"email"}, Unique: true},
		},
	}
	if schema.Collection != "users" {
		t.Error("unexpected collection")
	}
	if len(schema.Indexes) != 1 || !schema.Indexes[0].Unique {
		t.Error("unexpected index")
	}
}
