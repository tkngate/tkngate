package limiter

import (
	"testing"
	"time"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

func TestRateLimiter(t *testing.T) {
	// Setup logger and config
	logging.InitLogger()
	config.Cfg = config.Config{}
	config.Cfg.RateLimit.Enabled = true
	config.Cfg.RateLimit.RequestsPerMinute = 60
	config.Cfg.RateLimit.BurstSize = 2

	GlobalManager.Reset()

	sessionID := "test-session"

	// Request 1: allowed (burst 2)
	if !GlobalManager.Allow(sessionID) {
		t.Errorf("Expected Request 1 to be allowed")
	}

	// Request 2: allowed (burst 2)
	if !GlobalManager.Allow(sessionID) {
		t.Errorf("Expected Request 2 to be allowed")
	}

	// Request 3: blocked (bucket empty)
	if GlobalManager.Allow(sessionID) {
		t.Errorf("Expected Request 3 to be blocked")
	}

	// Sleep for 1 second (1 token should be refilled at 60 RPM)
	time.Sleep(1 * time.Second)

	// Request 4: allowed (bucket refilled 1 token)
	if !GlobalManager.Allow(sessionID) {
		t.Errorf("Expected Request 4 to be allowed after 1s sleep")
	}

	// Request 5: blocked (bucket empty again)
	if GlobalManager.Allow(sessionID) {
		t.Errorf("Expected Request 5 to be blocked")
	}
}
