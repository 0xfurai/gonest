package sql

import (
	"testing"
	"time"
)

func TestBuildDSN_Postgres(t *testing.T) {
	opts := Options{
		Driver:   DriverPostgres,
		Host:     "localhost",
		Port:     5432,
		User:     "myuser",
		Password: "mypass",
		Database: "mydb",
		SSLMode:  "disable",
	}

	dsn := opts.BuildDSN()
	expected := "host=localhost port=5432 user=myuser password=mypass dbname=mydb sslmode=disable"
	if dsn != expected {
		t.Errorf("expected %q, got %q", expected, dsn)
	}
}

func TestBuildDSN_Postgres_DefaultPort(t *testing.T) {
	opts := Options{
		Driver:   DriverPostgres,
		Host:     "db.example.com",
		User:     "admin",
		Password: "secret",
		Database: "prod",
	}

	dsn := opts.BuildDSN()
	expected := "host=db.example.com port=5432 user=admin password=secret dbname=prod sslmode=disable"
	if dsn != expected {
		t.Errorf("expected %q, got %q", expected, dsn)
	}
}

func TestBuildDSN_MySQL(t *testing.T) {
	opts := Options{
		Driver:   DriverMySQL,
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "secret",
		Database: "myapp",
	}

	dsn := opts.BuildDSN()
	expected := "root:secret@tcp(localhost:3306)/myapp?parseTime=true"
	if dsn != expected {
		t.Errorf("expected %q, got %q", expected, dsn)
	}
}

func TestBuildDSN_MySQL_DefaultPort(t *testing.T) {
	opts := Options{
		Driver:   DriverMySQL,
		Host:     "mysql.local",
		User:     "app",
		Password: "pass",
		Database: "db",
	}

	dsn := opts.BuildDSN()
	if dsn != "app:pass@tcp(mysql.local:3306)/db?parseTime=true" {
		t.Errorf("unexpected DSN: %q", dsn)
	}
}

func TestBuildDSN_MySQL_WithParams(t *testing.T) {
	opts := Options{
		Driver:   DriverMySQL,
		Host:     "localhost",
		User:     "root",
		Password: "",
		Database: "test",
		Params:   map[string]string{"charset": "utf8mb4"},
	}

	dsn := opts.BuildDSN()
	// Should contain both parseTime and charset
	if !containsSubstring(dsn, "parseTime=true") {
		t.Errorf("expected parseTime, got %q", dsn)
	}
	if !containsSubstring(dsn, "charset=utf8mb4") {
		t.Errorf("expected charset, got %q", dsn)
	}
}

func TestBuildDSN_SQLite_Memory(t *testing.T) {
	opts := Options{Driver: DriverSQLite}

	dsn := opts.BuildDSN()
	if dsn != ":memory:" {
		t.Errorf("expected ':memory:', got %q", dsn)
	}
}

func TestBuildDSN_SQLite_File(t *testing.T) {
	opts := Options{
		Driver:   DriverSQLite,
		Database: "./data.db",
	}

	dsn := opts.BuildDSN()
	if dsn != "./data.db" {
		t.Errorf("expected './data.db', got %q", dsn)
	}
}

func TestBuildDSN_SQLite_WithParams(t *testing.T) {
	opts := Options{
		Driver:   DriverSQLite,
		Database: "./data.db",
		Params:   map[string]string{"_journal_mode": "WAL"},
	}

	dsn := opts.BuildDSN()
	if !containsSubstring(dsn, "./data.db?") || !containsSubstring(dsn, "_journal_mode=WAL") {
		t.Errorf("unexpected DSN: %q", dsn)
	}
}

func TestBuildDSN_SQLServer(t *testing.T) {
	opts := Options{
		Driver:   DriverSQLServer,
		Host:     "localhost",
		Port:     1433,
		User:     "sa",
		Password: "pass",
		Database: "master",
	}

	dsn := opts.BuildDSN()
	expected := "sqlserver://sa:pass@localhost:1433?database=master"
	if dsn != expected {
		t.Errorf("expected %q, got %q", expected, dsn)
	}
}

func TestBuildDSN_SQLServer_DefaultPort(t *testing.T) {
	opts := Options{
		Driver:   DriverSQLServer,
		Host:     "db-server",
		User:     "admin",
		Password: "secret",
		Database: "mydb",
	}

	dsn := opts.BuildDSN()
	if !containsSubstring(dsn, ":1433") {
		t.Errorf("expected default port 1433, got %q", dsn)
	}
}

func TestBuildDSN_RawDSN(t *testing.T) {
	raw := "postgres://user:pass@host:5432/db?sslmode=require"
	opts := Options{
		Driver: DriverPostgres,
		DSN:    raw,
		Host:   "should-be-ignored",
	}

	dsn := opts.BuildDSN()
	if dsn != raw {
		t.Errorf("expected raw DSN, got %q", dsn)
	}
}

func TestBuildDSN_DefaultDriver(t *testing.T) {
	opts := Options{
		Host:     "localhost",
		User:     "test",
		Password: "test",
		Database: "test",
	}

	// No driver specified -> defaults to postgres format
	dsn := opts.BuildDSN()
	if !containsSubstring(dsn, "host=localhost") {
		t.Errorf("expected postgres format for default driver, got %q", dsn)
	}
}

func TestOptions_PoolSettings(t *testing.T) {
	opts := Options{
		MaxOpenConns:    25,
		MaxIdleConns:    5,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}

	if opts.MaxOpenConns != 25 {
		t.Errorf("expected 25, got %d", opts.MaxOpenConns)
	}
	if opts.ConnMaxLifetime != 5*time.Minute {
		t.Errorf("expected 5m, got %v", opts.ConnMaxLifetime)
	}
}

func TestPostgres_WithExtraParams(t *testing.T) {
	opts := Options{
		Driver:   DriverPostgres,
		Host:     "localhost",
		User:     "user",
		Password: "pass",
		Database: "db",
		Params:   map[string]string{"application_name": "myapp"},
	}

	dsn := opts.BuildDSN()
	if !containsSubstring(dsn, "application_name=myapp") {
		t.Errorf("expected extra param, got %q", dsn)
	}
}

func containsSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
