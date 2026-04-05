package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"sync"
)

// ConfigService provides access to configuration values from environment
// variables and optional .env files.
type ConfigService struct {
	mu     sync.RWMutex
	values map[string]string
}

// NewConfigService creates a new ConfigService.
// If envFilePath is non-empty, it loads values from that .env file.
func NewConfigService(envFilePath string) *ConfigService {
	svc := &ConfigService{
		values: make(map[string]string),
	}

	// Load .env file if specified
	if envFilePath != "" {
		svc.loadEnvFile(envFilePath)
	}

	return svc
}

// Get returns a configuration value by key. Falls back to OS environment.
func (s *ConfigService) Get(key string) string {
	s.mu.RLock()
	if val, ok := s.values[key]; ok {
		s.mu.RUnlock()
		return val
	}
	s.mu.RUnlock()
	return os.Getenv(key)
}

// GetOrDefault returns a configuration value or the default if not found.
func (s *ConfigService) GetOrDefault(key, defaultValue string) string {
	val := s.Get(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// GetInt returns a configuration value as an int.
func (s *ConfigService) GetInt(key string) (int, error) {
	return strconv.Atoi(s.Get(key))
}

// GetIntOrDefault returns a configuration value as an int with a default.
func (s *ConfigService) GetIntOrDefault(key string, defaultValue int) int {
	val, err := s.GetInt(key)
	if err != nil {
		return defaultValue
	}
	return val
}

// GetBool returns a configuration value as a bool.
func (s *ConfigService) GetBool(key string) (bool, error) {
	return strconv.ParseBool(s.Get(key))
}

// GetBoolOrDefault returns a configuration value as a bool with a default.
func (s *ConfigService) GetBoolOrDefault(key string, defaultValue bool) bool {
	val, err := s.GetBool(key)
	if err != nil {
		return defaultValue
	}
	return val
}

// Has checks if a configuration key exists.
func (s *ConfigService) Has(key string) bool {
	s.mu.RLock()
	_, ok := s.values[key]
	s.mu.RUnlock()
	if ok {
		return true
	}
	_, ok = os.LookupEnv(key)
	return ok
}

// Set sets a configuration value at runtime.
func (s *ConfigService) Set(key, value string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.values[key] = value
}

func (s *ConfigService) loadEnvFile(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])
		// Remove surrounding quotes
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}
		s.values[key] = value
	}
}
