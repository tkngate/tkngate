package budget

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
	"strconv"

	_ "modernc.org/sqlite"
	"tkngate/internal/cluster"
	"tkngate/internal/config"
)

type Ledger struct {
	db *sql.DB
	mu sync.RWMutex
}

var GlobalLedger *Ledger

// DB exposes the underlying database handle for shared use (e.g., mesh reputation tables).
func (l *Ledger) DB() *sql.DB {
	return l.db
}

var DatabaseName = "budget.db"

func InitLedger() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	dir := filepath.Join(homeDir, ".tkngate")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}

	dbPath := filepath.Join(dir, DatabaseName)
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return err
	}
	
	// Enable Write-Ahead Logging (WAL) for high concurrency
	if _, err := db.Exec(`PRAGMA journal_mode=WAL; PRAGMA synchronous=NORMAL;`); err != nil {
		return err
	}

	// Create tables
	query := `
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT DEFAULT '',
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		estimated_cost_usd REAL DEFAULT 0.0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS tkngate_sessions (
		session_id TEXT PRIMARY KEY,
		allocated_budget_usd REAL NOT NULL,
		consumed_budget_usd REAL DEFAULT 0.0,
		current_state TEXT DEFAULT 'GREEN',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS token_pool_nodes (
		node_id TEXT PRIMARY KEY,
		provider_type TEXT NOT NULL,
		blinded_key_hash TEXT NOT NULL,
		measured_tpm_limit INTEGER NOT NULL,
		remaining_tokens_quota INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tkngate_organizations (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT UNIQUE NOT NULL,
		budget_limit_usd REAL NOT NULL,
		consumed_usd REAL DEFAULT 0.0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS tkngate_virtual_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_hash TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		allocated_budget_usd REAL NOT NULL,
		consumed_budget_usd REAL DEFAULT 0.0,
		org_id INTEGER DEFAULT 0,
		allowed_providers TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		return err
	}

	// Add session_id to existing table if migrating from v0.0.1
	db.Exec(`ALTER TABLE transactions ADD COLUMN session_id TEXT DEFAULT ''`)
	
	// Migrate existing virtual keys for v2.0.0 RBAC
	db.Exec(`ALTER TABLE tkngate_virtual_keys ADD COLUMN org_id INTEGER DEFAULT 0`)
	db.Exec(`ALTER TABLE tkngate_virtual_keys ADD COLUMN allowed_providers TEXT DEFAULT ''`)

	// v2.6.0: Add virtual_key_hash to transactions and current_state to virtual keys for alerting
	db.Exec(`ALTER TABLE transactions ADD COLUMN virtual_key_hash TEXT DEFAULT ''`)
	db.Exec(`ALTER TABLE tkngate_virtual_keys ADD COLUMN current_state TEXT DEFAULT 'GREEN'`)

	GlobalLedger = &Ledger{db: db}
	return nil
}

func (l *Ledger) RecordTransaction(tx Transaction) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if config.Cfg.Cluster.Enabled && cluster.RedisClient != nil {
		ctx := context.Background()
		// Write to Redis for shared state
		cluster.RedisClient.IncrByFloat(ctx, "tkngate:budget:global_spend", tx.EstimatedCostUSD)
		if tx.SessionID != "" {
			cluster.RedisClient.IncrByFloat(ctx, "tkngate:budget:session:"+tx.SessionID, tx.EstimatedCostUSD)
		}
		if tx.VirtualKeyHash != "" {
			cluster.RedisClient.IncrByFloat(ctx, "tkngate:budget:vkey:"+tx.VirtualKeyHash, tx.EstimatedCostUSD)
		}
	}

	query := `INSERT INTO transactions (session_id, virtual_key_hash, provider, model, input_tokens, output_tokens, estimated_cost_usd, timestamp) 
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?)`

	_, err := l.db.Exec(query, tx.SessionID, tx.VirtualKeyHash, tx.Provider, tx.Model, tx.InputTokens, tx.OutputTokens, tx.EstimatedCostUSD, time.Now())
	if err != nil {
		return err
	}

	if tx.SessionID != "" {
		_, err = l.db.Exec(`UPDATE tkngate_sessions SET consumed_budget_usd = consumed_budget_usd + ? WHERE session_id = ?`, tx.EstimatedCostUSD, tx.SessionID)
	}

	if tx.VirtualKeyHash != "" {
		_, err = l.db.Exec(`UPDATE tkngate_virtual_keys SET consumed_budget_usd = consumed_budget_usd + ? WHERE key_hash = ?`, tx.EstimatedCostUSD, tx.VirtualKeyHash)
		// Update organization budget if applicable
		_, err = l.db.Exec(`UPDATE tkngate_organizations SET consumed_usd = consumed_usd + ? WHERE id = (SELECT org_id FROM tkngate_virtual_keys WHERE key_hash = ?)`, tx.EstimatedCostUSD, tx.VirtualKeyHash)
	}

	return err
}

func (l *Ledger) GetTotalSpend() (float64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if config.Cfg.Cluster.Enabled && cluster.RedisClient != nil {
		val, err := cluster.RedisClient.Get(context.Background(), "tkngate:budget:global_spend").Result()
		if err == nil {
			return strconv.ParseFloat(val, 64)
		}
	}

	var total float64
	err := l.db.QueryRow(`SELECT COALESCE(SUM(estimated_cost_usd), 0.0) FROM transactions`).Scan(&total)
	return total, err
}

func (l *Ledger) Reset() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.db.Exec(`DELETE FROM transactions`)
	if err == nil {
		l.db.Exec(`DELETE FROM tkngate_sessions`)
	}
	return err
}

func (l *Ledger) GetSessionSpend(sessionID string) (float64, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	if config.Cfg.Cluster.Enabled && cluster.RedisClient != nil {
		val, err := cluster.RedisClient.Get(context.Background(), "tkngate:budget:session:"+sessionID).Result()
		if err == nil {
			return strconv.ParseFloat(val, 64)
		}
	}

	var total float64
	err := l.db.QueryRow(`SELECT COALESCE(SUM(estimated_cost_usd), 0.0) FROM transactions WHERE session_id = ?`, sessionID).Scan(&total)
	return total, err
}

func (l *Ledger) EnsureSession(sessionID string, allocatedBudget float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.db.Exec(`INSERT OR IGNORE INTO tkngate_sessions (session_id, allocated_budget_usd) VALUES (?, ?)`, sessionID, allocatedBudget)
	return err
}

func (l *Ledger) AddPoolNode(node PoolNode) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	query := `INSERT INTO token_pool_nodes (node_id, provider_type, blinded_key_hash, measured_tpm_limit, remaining_tokens_quota) VALUES (?, ?, ?, ?, ?)`
	_, err := l.db.Exec(query, node.NodeID, node.ProviderType, node.BlindedKeyHash, node.MeasuredTpmLimit, node.RemainingTokensQuota)
	return err
}

func (l *Ledger) RemovePoolNode(nodeID string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	res, err := l.db.Exec(`DELETE FROM token_pool_nodes WHERE node_id = ?`, nodeID)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (l *Ledger) GetPoolNodes(provider string) ([]PoolNode, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	rows, err := l.db.Query(`SELECT node_id, provider_type, blinded_key_hash, measured_tpm_limit, remaining_tokens_quota FROM token_pool_nodes WHERE provider_type = ?`, provider)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes = []PoolNode{}
	for rows.Next() {
		var n PoolNode
		if err := rows.Scan(&n.NodeID, &n.ProviderType, &n.BlindedKeyHash, &n.MeasuredTpmLimit, &n.RemainingTokensQuota); err == nil {
			nodes = append(nodes, n)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return nodes, nil
}

func (l *Ledger) GetSessions() ([]SessionState, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	rows, err := l.db.Query(`SELECT session_id, allocated_budget_usd, consumed_budget_usd, current_state, created_at FROM tkngate_sessions`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sessions = []SessionState{}
	for rows.Next() {
		var s SessionState
		if err := rows.Scan(&s.SessionID, &s.AllocatedBudget, &s.ConsumedBudget, &s.CurrentState, &s.CreatedAt); err == nil {
			sessions = append(sessions, s)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}

func (l *Ledger) GetTransactionCount() (int, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	var count int
	err := l.db.QueryRow(`SELECT COUNT(*) FROM transactions`).Scan(&count)
	return count, err
}

func (l *Ledger) DecrementPoolQuota(nodeID string, tokens int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.db.Exec(`UPDATE token_pool_nodes SET remaining_tokens_quota = remaining_tokens_quota - ? WHERE node_id = ? AND remaining_tokens_quota > 0`, tokens, nodeID)
}

// v1.3.0: Virtual Key Management

type VirtualKeyRecord struct {
	ID               int        `json:"id"`
	KeyHash          string     `json:"key_hash"`
	Name             string     `json:"name"`
	AllocatedBudget  float64    `json:"allocated_budget_usd"`
	ConsumedBudget   float64    `json:"consumed_budget_usd"`
	OrgID            int        `json:"org_id"`
	AllowedProviders string     `json:"allowed_providers"`
	CurrentState     BudgetZone `json:"current_state"`
	CreatedAt        string     `json:"created_at"`
}

type OrganizationRecord struct {
	ID             int
	Name           string
	BudgetLimitUSD float64
	ConsumedUSD    float64
	CreatedAt      string
}

func (l *Ledger) CreateOrganization(name string, limit float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.db.Exec(`INSERT INTO tkngate_organizations (name, budget_limit_usd) VALUES (?, ?)`, name, limit)
	return err
}

func (l *Ledger) DeleteOrganization(id int) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.db.Exec(`DELETE FROM tkngate_organizations WHERE id = ?`, id)
	return err
}

func (l *Ledger) GetOrganizations() ([]OrganizationRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	rows, err := l.db.Query(`SELECT id, name, budget_limit_usd, consumed_usd, created_at FROM tkngate_organizations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var orgs []OrganizationRecord
	for rows.Next() {
		var o OrganizationRecord
		if err := rows.Scan(&o.ID, &o.Name, &o.BudgetLimitUSD, &o.ConsumedUSD, &o.CreatedAt); err == nil {
			orgs = append(orgs, o)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return orgs, nil
}

func (l *Ledger) RegisterVirtualKey(keyHash, name string, allocatedBudget float64, orgID int, allowedProviders string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.db.Exec(`INSERT INTO tkngate_virtual_keys (key_hash, name, allocated_budget_usd, org_id, allowed_providers) VALUES (?, ?, ?, ?, ?)`, keyHash, name, allocatedBudget, orgID, allowedProviders)
	return err
}

func (l *Ledger) GetVirtualKeys() ([]VirtualKeyRecord, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	rows, err := l.db.Query(`SELECT id, key_hash, name, allocated_budget_usd, consumed_budget_usd, org_id, allowed_providers, current_state, created_at FROM tkngate_virtual_keys`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []VirtualKeyRecord
	for rows.Next() {
		var k VirtualKeyRecord
		if err := rows.Scan(&k.ID, &k.KeyHash, &k.Name, &k.AllocatedBudget, &k.ConsumedBudget, &k.OrgID, &k.AllowedProviders, &k.CurrentState, &k.CreatedAt); err == nil {
			keys = append(keys, k)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return keys, nil
}

func (l *Ledger) ChargeVirtualKey(keyHash string, amount float64) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.db.Exec(`UPDATE tkngate_virtual_keys SET consumed_budget_usd = consumed_budget_usd + ? WHERE key_hash = ?`, amount, keyHash)
	return err
}

func (l *Ledger) UpdateVirtualKeyZone(keyHash string, zone BudgetZone) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	_, err := l.db.Exec(`UPDATE tkngate_virtual_keys SET current_state = ? WHERE key_hash = ?`, zone, keyHash)
	return err
}

type DailySpend struct {
	Date     string  `json:"date"`
	SpendUSD float64 `json:"spend_usd"`
}

func (l *Ledger) GetDailySpend(days int) ([]DailySpend, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	
	// Group transactions by date for the last N days
	query := `
		SELECT DATE(timestamp) as d, SUM(estimated_cost_usd) 
		FROM transactions 
		WHERE timestamp >= date('now', ?) 
		GROUP BY d 
		ORDER BY d ASC
	`
	rows, err := l.db.Query(query, fmt.Sprintf("-%d days", days))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var daily []DailySpend
	for rows.Next() {
		var d DailySpend
		if err := rows.Scan(&d.Date, &d.SpendUSD); err == nil {
			daily = append(daily, d)
		}
	}
	return daily, rows.Err()
}

func (l *Ledger) GetTransactions(keyHash string, limit int) ([]Transaction, error) {
	l.mu.RLock()
	defer l.mu.RUnlock()

	query := `
		SELECT id, session_id, virtual_key_hash, provider, model, input_tokens, output_tokens, estimated_cost_usd, timestamp 
		FROM transactions 
	`
	var rows *sql.Rows
	var err error

	if keyHash != "" {
		query += `WHERE virtual_key_hash = ? ORDER BY timestamp DESC LIMIT ?`
		rows, err = l.db.Query(query, keyHash, limit)
	} else {
		query += `ORDER BY timestamp DESC LIMIT ?`
		rows, err = l.db.Query(query, limit)
	}

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var txs []Transaction
	for rows.Next() {
		var t Transaction
		if err := rows.Scan(&t.ID, &t.SessionID, &t.VirtualKeyHash, &t.Provider, &t.Model, &t.InputTokens, &t.OutputTokens, &t.EstimatedCostUSD, &t.Timestamp); err == nil {
			txs = append(txs, t)
		}
	}
	return txs, rows.Err()
}

func (l *Ledger) RevokeVirtualKey(name string) error {
	l.mu.Lock()
	defer l.mu.Unlock()
	res, err := l.db.Exec(`DELETE FROM tkngate_virtual_keys WHERE name = ?`, name)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// InitMemoryLedger creates an in-memory ledger strictly for unit testing.
func InitMemoryLedger() error {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		return err
	}

	query := `
	CREATE TABLE IF NOT EXISTS transactions (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		session_id TEXT DEFAULT '',
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		input_tokens INTEGER DEFAULT 0,
		output_tokens INTEGER DEFAULT 0,
		estimated_cost_usd REAL DEFAULT 0.0,
		timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS tkngate_sessions (
		session_id TEXT PRIMARY KEY,
		allocated_budget_usd REAL NOT NULL,
		consumed_budget_usd REAL DEFAULT 0.0,
		current_state TEXT DEFAULT 'GREEN',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	
	CREATE TABLE IF NOT EXISTS token_pool_nodes (
		node_id TEXT PRIMARY KEY,
		provider_type TEXT NOT NULL,
		blinded_key_hash TEXT NOT NULL,
		measured_tpm_limit INTEGER NOT NULL,
		remaining_tokens_quota INTEGER NOT NULL
	);

	CREATE TABLE IF NOT EXISTS tkngate_virtual_keys (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		key_hash TEXT UNIQUE NOT NULL,
		name TEXT NOT NULL,
		allocated_budget_usd REAL NOT NULL,
		consumed_budget_usd REAL DEFAULT 0.0,
		current_state TEXT DEFAULT 'GREEN',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`
	if _, err := db.Exec(query); err != nil {
		return err
	}

	db.Exec(`ALTER TABLE transactions ADD COLUMN virtual_key_hash TEXT DEFAULT ''`)

	GlobalLedger = &Ledger{db: db}
	return nil
}
