package limiter

import (
	"sync"
	"time"

	"golang.org/x/time/rate"
	"tkngate/internal/config"
	"tkngate/internal/logging"
)

// limiterEntry wraps a rate.Limiter with a timestamp for cleanup.
type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Manager manages isolated rate limiters for individual sessions/keys
type Manager struct {
	mu       sync.RWMutex
	limiters map[string]*limiterEntry
}

var GlobalManager = &Manager{
	limiters: make(map[string]*limiterEntry),
}

// StartCleanup launches a background goroutine that evicts stale limiters
// every 5 minutes. Limiters unused for more than 30 minutes are removed.
func (m *Manager) StartCleanup() {
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			m.mu.Lock()
			now := time.Now()
			evicted := 0
			for id, entry := range m.limiters {
				if now.Sub(entry.lastSeen) > 30*time.Minute {
					delete(m.limiters, id)
					evicted++
				}
			}
			m.mu.Unlock()
			if evicted > 0 {
				logging.Logger.Info("Rate limiter cleanup", "evicted", evicted, "remaining", len(m.limiters))
			}
		}
	}()
}

// Allow checks if the given sessionID/keyHash is allowed to make a request
func (m *Manager) Allow(identifier string) bool {
	if !config.Cfg.RateLimit.Enabled {
		return true // Rate limiting is disabled globally
	}

	m.mu.RLock()
	entry, exists := m.limiters[identifier]
	m.mu.RUnlock()

	if exists {
		m.mu.Lock()
		entry.lastSeen = time.Now()
		m.mu.Unlock()
		return entry.limiter.Allow()
	}

	// Create a new limiter for this identifier
	m.mu.Lock()
	// Double-check locking
	entry, exists = m.limiters[identifier]
	if !exists {
		// Convert RPM to requests per second
		r := rate.Limit(float64(config.Cfg.RateLimit.RequestsPerMinute) / 60.0)
		b := config.Cfg.RateLimit.BurstSize

		lim := rate.NewLimiter(r, b)
		entry = &limiterEntry{limiter: lim, lastSeen: time.Now()}
		m.limiters[identifier] = entry
		logging.Logger.Info("Created new rate limiter", "identifier", identifier, "rpm", config.Cfg.RateLimit.RequestsPerMinute, "burst", b)
	}
	m.mu.Unlock()

	return entry.limiter.Allow()
}

// Reset clears all limiters (useful for testing or config reloads)
func (m *Manager) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.limiters = make(map[string]*limiterEntry)
}
