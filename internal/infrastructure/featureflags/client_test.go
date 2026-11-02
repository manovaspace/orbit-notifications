package featureflags

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/open-feature/go-sdk/openfeature"
)

// recorder collects log messages emitted by NewFromEnv. NewFromEnv logs from the
// caller's goroutine only, so no locking is needed to read the result, but -race
// stays clean if that ever changes.
type recorder struct {
	mu   sync.Mutex
	msgs []string
}

func (r *recorder) log(msg string, _ ...any) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.msgs = append(r.msgs, msg)
}

func (r *recorder) contains(want string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, msg := range r.msgs {
		if strings.Contains(msg, want) {
			return true
		}
	}
	return false
}

func TestNewFromEnvUnreachableFallsBackToDefaults(t *testing.T) {
	t.Setenv("UNLEASH_URL", "http://10.255.255.1:1/api/")
	t.Cleanup(openfeature.Shutdown)

	rec := &recorder{}
	start := time.Now()
	evaluator := NewFromEnv("orbit-notifications-test", rec.log)
	elapsed := time.Since(start)

	if elapsed >= time.Second {
		t.Fatalf("NewFromEnv took %s, want under 1s", elapsed)
	}
	if got := evaluator.Bool(context.Background(), "x", true); got != true {
		t.Fatalf("Bool(x, true) = %v, want true", got)
	}
}

func TestNewFromEnvDisabledURLReturnsDefaultsImmediately(t *testing.T) {
	for _, url := range []string{"off", "OFF", "disabled", "Disabled"} {
		t.Run(url, func(t *testing.T) {
			t.Setenv("UNLEASH_URL", url)
			t.Cleanup(openfeature.Shutdown)

			rec := &recorder{}
			start := time.Now()
			evaluator := NewFromEnv("orbit-notifications-test", rec.log)
			elapsed := time.Since(start)

			if elapsed >= time.Second {
				t.Fatalf("NewFromEnv took %s, want immediate return", elapsed)
			}
			if got := evaluator.Bool(context.Background(), "x", true); got != true {
				t.Fatalf("Bool(x, true) = %v, want true", got)
			}
			if !rec.contains("feature_flags_disabled") {
				t.Fatalf("expected feature_flags_disabled log, got %v", rec.msgs)
			}
		})
	}
}
