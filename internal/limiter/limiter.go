package limiter

import (
	"sync"
	"golang.org/x/time/rate"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// GlobalManager manages isolated rate limiters for individual sessions/keys
type Manager struct {
	mu       sync.RWMutex
	limiters map[string]*rate.Limiter
}

var GlobalManager = &Manager{
	limiters: make(map[string]*rate.Limiter),
}

// Allow checks if the given sessionID/keyHash is allowed to make a request
func (m *Manager) Allow(identifier string) bool {
	if !config.Cfg.RateLimit.Enabled {
		return true // Rate limiting is disabled globally
	}

	m.mu.RLock()
	limiter, exists := m.limiters[identifier]
	m.mu.RUnlock()

	if !exists {
		// Create a new limiter for this identifier
		m.mu.Lock()
		// Double-check locking
		limiter, exists = m.limiters[identifier]
		if !exists {
			// Convert RPM to requests per second
			r := rate.Limit(float64(config.Cfg.RateLimit.RequestsPerMinute) / 60.0)
			b := config.Cfg.RateLimit.BurstSize
			
			limiter = rate.NewLimiter(r, b)
			m.limiters[identifier] = limiter
			logging.Logger.Info("Created new rate limiter", "identifier", identifier, "rpm", config.Cfg.RateLimit.RequestsPerMinute, "burst", b)
		}
		m.mu.Unlock()
	}

	return limiter.Allow()
}

// Reset clears all limiters (useful for testing or config reloads)
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters = make(map[string]*rate.Limiter)
}
