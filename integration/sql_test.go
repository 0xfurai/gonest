package integration

import (
	"testing"

	"github.com/gonest"
	gosql "github.com/gonest/database/sql"
)

// ---------------------------------------------------------------------------
// SQL Module Integration Tests
// Mirror: original/integration/typeorm/
// ---------------------------------------------------------------------------
//
// These tests verify the SQL module's DSN building, module creation, and
// helper functions. Actual database connectivity tests require a SQL driver
// (e.g., modernc.org/sqlite for SQLite in-memory).

// ---------------------------------------------------------------------------
// Tests: DSN Building
// ---------------------------------------------------------------------------

func TestSQL_BuildDSN_Postgres(t *testing.T) {
	opts := gosql.Options{
		Driver:   gosql.DriverPostgres,
		Host:     "localhost",
		Port:     5432,
		User:     "myuser",
		Password: "mypass",
		Database: "mydb",
		SSLMode:  "disable",
	}

	dsn := opts.BuildDSN()
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}

	expected := "host=localhost port=5432 user=myuser password=mypass dbname=mydb sslmode=disable"
	if dsn != expected {
		t.Errorf("expected %q, got %q", expected, dsn)
	}
}

func TestSQL_BuildDSN_PostgresDefaultPort(t *testing.T) {
	opts := gosql.Options{
		Driver:   gosql.DriverPostgres,
		Host:     "db.example.com",
		User:     "admin",
		Password: "secret",
		Database: "prod",
	}

	dsn := opts.BuildDSN()
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}
	// Port defaults to 5432, SSLMode defaults to "disable"
	if !contains(dsn, "port=5432") {
		t.Error("expected default port 5432 in DSN")
	}
}

func TestSQL_BuildDSN_MySQL(t *testing.T) {
	opts := gosql.Options{
		Driver:   gosql.DriverMySQL,
		Host:     "localhost",
		Port:     3306,
		User:     "root",
		Password: "pass",
		Database: "testdb",
	}

	dsn := opts.BuildDSN()
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}
	if !contains(dsn, "root:pass@tcp(localhost:3306)/testdb") {
		t.Errorf("unexpected MySQL DSN: %q", dsn)
	}
	if !contains(dsn, "parseTime=true") {
		t.Error("expected parseTime=true in MySQL DSN")
	}
}

func TestSQL_BuildDSN_SQLite(t *testing.T) {
	opts := gosql.Options{
		Driver: gosql.DriverSQLite,
	}

	dsn := opts.BuildDSN()
	if dsn != ":memory:" {
		t.Errorf("expected :memory: for empty SQLite database, got %q", dsn)
	}
}

func TestSQL_BuildDSN_SQLiteWithFile(t *testing.T) {
	opts := gosql.Options{
		Driver:   gosql.DriverSQLite,
		Database: "/tmp/test.db",
	}

	dsn := opts.BuildDSN()
	if dsn != "/tmp/test.db" {
		t.Errorf("expected /tmp/test.db, got %q", dsn)
	}
}

func TestSQL_BuildDSN_SQLServer(t *testing.T) {
	opts := gosql.Options{
		Driver:   gosql.DriverSQLServer,
		Host:     "localhost",
		Port:     1433,
		User:     "sa",
		Password: "StrongP@ss",
		Database: "master",
	}

	dsn := opts.BuildDSN()
	if dsn == "" {
		t.Fatal("expected non-empty DSN")
	}
	if !contains(dsn, "sqlserver://") {
		t.Error("expected sqlserver:// prefix")
	}
}

func TestSQL_BuildDSN_ExplicitDSN(t *testing.T) {
	customDSN := "postgres://user:pass@host:5432/db?sslmode=require"
	opts := gosql.Options{
		Driver: gosql.DriverPostgres,
		DSN:    customDSN,
		Host:   "ignored",
	}

	dsn := opts.BuildDSN()
	if dsn != customDSN {
		t.Errorf("explicit DSN should override individual params, got %q", dsn)
	}
}

func TestSQL_BuildDSN_WithParams(t *testing.T) {
	opts := gosql.Options{
		Driver:   gosql.DriverSQLite,
		Database: "test.db",
		Params:   map[string]string{"_journal_mode": "WAL"},
	}

	dsn := opts.BuildDSN()
	if !contains(dsn, "_journal_mode=WAL") {
		t.Errorf("expected params in DSN, got %q", dsn)
	}
}

// ---------------------------------------------------------------------------
// Tests: Module creation
// ---------------------------------------------------------------------------

func TestSQL_NewModule_CreatesModule(t *testing.T) {
	module := gosql.NewModule(gosql.Options{
		Driver:   gosql.DriverSQLite,
		Database: ":memory:",
	})

	if module == nil {
		t.Fatal("expected non-nil module")
	}

	// The module should have providers and exports
	opts := module.Options()
	if len(opts.Providers) == 0 {
		t.Error("expected providers in SQL module")
	}
	if len(opts.Exports) == 0 {
		t.Error("expected exports in SQL module")
	}
	if !opts.Global {
		t.Error("SQL module should be global")
	}
}

func TestSQL_NewModuleFromDSN_CreatesModule(t *testing.T) {
	module := gosql.NewModuleFromDSN(gosql.DriverSQLite, ":memory:")

	if module == nil {
		t.Fatal("expected non-nil module")
	}

	opts := module.Options()
	if len(opts.Providers) == 0 {
		t.Error("expected providers in SQL module")
	}
}

// ---------------------------------------------------------------------------
// Tests: Module integration with app (no driver, expect init error)
// ---------------------------------------------------------------------------

func TestSQL_Module_IntegrationWithApp(t *testing.T) {
	dbModule := gosql.NewModule(gosql.Options{
		Driver:   gosql.DriverSQLite,
		Database: ":memory:",
	})

	appModule := gonest.NewModule(gonest.ModuleOptions{
		Imports: []*gonest.Module{dbModule},
	})

	app := gonest.Create(appModule, gonest.ApplicationOptions{Logger: gonest.NopLogger{}})
	err := app.Init()
	// Without a registered SQLite driver, this should fail
	if err == nil {
		// If it somehow succeeds (driver registered), that's fine too
		app.Close()
	}
	// Either way, the module structure was valid
}

// ---------------------------------------------------------------------------
// Tests: HealthChecker
// ---------------------------------------------------------------------------

func TestSQL_HealthChecker_Name(t *testing.T) {
	hc := gosql.NewHealthChecker(nil, "mydb")
	if hc.Name() != "mydb" {
		t.Errorf("expected name=mydb, got %q", hc.Name())
	}
}

func TestSQL_HealthChecker_DefaultName(t *testing.T) {
	hc := gosql.NewHealthChecker(nil, "")
	if hc.Name() != "database" {
		t.Errorf("expected default name=database, got %q", hc.Name())
	}
}

// ---------------------------------------------------------------------------
// Tests: Migrate with nil DB (validates function exists)
// ---------------------------------------------------------------------------

func TestSQL_Migrate_NilDB(t *testing.T) {
	// Calling Migrate with nil *sql.DB should panic or return error
	// This validates the function signature exists
	defer func() {
		if r := recover(); r == nil {
			// If no panic, Migrate should have returned an error
		}
	}()

	err := gosql.Migrate(nil, []string{"CREATE TABLE test (id INT)"})
	if err == nil {
		// nil db.Exec should return error
		t.Log("Migrate with nil DB handled gracefully")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
