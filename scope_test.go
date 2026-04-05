package gonest

import "testing"

func TestScope_Values(t *testing.T) {
	if ScopeSingleton != 0 {
		t.Errorf("ScopeSingleton should be 0, got %d", ScopeSingleton)
	}
	if ScopeRequest != 1 {
		t.Errorf("ScopeRequest should be 1, got %d", ScopeRequest)
	}
	if ScopeTransient != 2 {
		t.Errorf("ScopeTransient should be 2, got %d", ScopeTransient)
	}
}
