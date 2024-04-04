package legacy_executor_verifier

import (
	"testing"
	"time"
)

func TestExecutor_SlowExecutor_Timeout(t *testing.T) {
	// this test exists only to make sure timeout is implemented, not to test the actual timeout duration
	executor, err := NewExecutor("http://localhost:8080", 100*time.Millisecond) // use non-responding url
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if err.Error() != "failed to dial grpc: context deadline exceeded" {
		t.Errorf("Unexpected error: %v", err)
	}

	executor.Close()
}
