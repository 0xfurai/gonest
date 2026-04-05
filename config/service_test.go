package config

import (
	"os"
	"testing"
)

func TestConfigService_Get_FromEnv(t *testing.T) {
	os.Setenv("TEST_CONFIG_KEY", "test_value")
	defer os.Unsetenv("TEST_CONFIG_KEY")

	svc := NewConfigService("")
	if svc.Get("TEST_CONFIG_KEY") != "test_value" {
		t.Errorf("expected 'test_value', got %q", svc.Get("TEST_CONFIG_KEY"))
	}
}

func TestConfigService_GetOrDefault(t *testing.T) {
	svc := NewConfigService("")
	val := svc.GetOrDefault("NONEXISTENT_KEY_12345", "fallback")
	if val != "fallback" {
		t.Errorf("expected 'fallback', got %q", val)
	}
}

func TestConfigService_Set(t *testing.T) {
	svc := NewConfigService("")
	svc.Set("MY_KEY", "my_value")
	if svc.Get("MY_KEY") != "my_value" {
		t.Errorf("expected 'my_value', got %q", svc.Get("MY_KEY"))
	}
}

func TestConfigService_Has(t *testing.T) {
	svc := NewConfigService("")
	svc.Set("EXISTS", "yes")

	if !svc.Has("EXISTS") {
		t.Error("expected Has to return true for existing key")
	}
	if svc.Has("DOES_NOT_EXIST_12345") {
		t.Error("expected Has to return false for missing key")
	}
}

func TestConfigService_GetInt(t *testing.T) {
	svc := NewConfigService("")
	svc.Set("PORT", "8080")

	val, err := svc.GetInt("PORT")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != 8080 {
		t.Errorf("expected 8080, got %d", val)
	}
}

func TestConfigService_GetInt_Invalid(t *testing.T) {
	svc := NewConfigService("")
	svc.Set("PORT", "not_a_number")

	_, err := svc.GetInt("PORT")
	if err == nil {
		t.Error("expected error for non-integer value")
	}
}

func TestConfigService_GetIntOrDefault(t *testing.T) {
	svc := NewConfigService("")
	val := svc.GetIntOrDefault("MISSING_PORT", 3000)
	if val != 3000 {
		t.Errorf("expected 3000, got %d", val)
	}
}

func TestConfigService_GetBool(t *testing.T) {
	svc := NewConfigService("")
	svc.Set("DEBUG", "true")

	val, err := svc.GetBool("DEBUG")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !val {
		t.Error("expected true")
	}
}

func TestConfigService_GetBoolOrDefault(t *testing.T) {
	svc := NewConfigService("")
	val := svc.GetBoolOrDefault("MISSING_DEBUG", false)
	if val {
		t.Error("expected false")
	}
}

func TestConfigService_LoadEnvFile(t *testing.T) {
	// Create a temp .env file
	content := `
# Comment line
DB_HOST=localhost
DB_PORT=5432
DB_NAME="mydb"
DB_PASS='secret'
EMPTY=
`
	tmpfile, err := os.CreateTemp("", "env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())
	tmpfile.WriteString(content)
	tmpfile.Close()

	svc := NewConfigService(tmpfile.Name())

	tests := []struct {
		key      string
		expected string
	}{
		{"DB_HOST", "localhost"},
		{"DB_PORT", "5432"},
		{"DB_NAME", "mydb"},
		{"DB_PASS", "secret"},
		{"EMPTY", ""},
	}

	for _, tt := range tests {
		val := svc.Get(tt.key)
		if val != tt.expected {
			t.Errorf("key %q: expected %q, got %q", tt.key, tt.expected, val)
		}
	}
}

func TestConfigService_LoadEnvFile_Missing(t *testing.T) {
	svc := NewConfigService("/nonexistent/file")
	// Should not panic, just ignore
	if svc.Get("ANY") != "" {
		t.Error("expected empty for missing file")
	}
}

func TestConfigService_SetOverridesEnv(t *testing.T) {
	os.Setenv("OVERRIDE_TEST", "from_env")
	defer os.Unsetenv("OVERRIDE_TEST")

	svc := NewConfigService("")
	svc.Set("OVERRIDE_TEST", "from_set")

	if svc.Get("OVERRIDE_TEST") != "from_set" {
		t.Errorf("expected 'from_set', got %q", svc.Get("OVERRIDE_TEST"))
	}
}
