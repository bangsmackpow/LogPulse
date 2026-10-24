package db

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	connStr := fmt.Sprintf("%s?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)", dbPath)
	dbConn, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	s := &Store{db: dbConn}
	if err := s.migrate(); err != nil {
		dbConn.Close()
		return nil, fmt.Errorf("migration failed: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *Store) migrate() error {
	schema := `
	CREATE TABLE IF NOT EXISTS discovered_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		file_path TEXT UNIQUE NOT NULL,
		app_name TEXT NOT NULL,
		file_size INTEGER NOT NULL,
		last_modified DATETIME NOT NULL,
		format TEXT NOT NULL,
		is_favorite INTEGER NOT NULL DEFAULT 0,
		tag TEXT NOT NULL DEFAULT ''
	);

	CREATE INDEX IF NOT EXISTS idx_discovered_logs_modified ON discovered_logs(last_modified DESC);
	CREATE INDEX IF NOT EXISTS idx_discovered_logs_app ON discovered_logs(app_name);

	CREATE TABLE IF NOT EXISTS ai_accounting (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp DATETIME NOT NULL,
		provider TEXT NOT NULL,
		model TEXT NOT NULL,
		input_tokens INTEGER NOT NULL,
		output_tokens INTEGER NOT NULL,
		cost_usd REAL NOT NULL,
		query_summary TEXT NOT NULL
	);

	CREATE INDEX IF NOT EXISTS idx_ai_accounting_timestamp ON ai_accounting(timestamp);

	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT NOT NULL
	);
	`
	_, err := s.db.Exec(schema)
	return err
}

func (s *Store) SaveDiscoveredLog(log DiscoveredLog) error {
	query := `
	INSERT INTO discovered_logs (file_path, app_name, file_size, last_modified, format, is_favorite, tag)
	VALUES (?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(file_path) DO UPDATE SET
		app_name = excluded.app_name,
		file_size = excluded.file_size,
		last_modified = excluded.last_modified,
		format = excluded.format;
	`
	isFav := 0
	if log.IsFavorite {
		isFav = 1
	}
	_, err := s.db.Exec(query, log.FilePath, log.AppName, log.FileSize, log.LastModified.Format(time.RFC3339), log.Format, isFav, log.Tag)
	return err
}

func (s *Store) ListDiscoveredLogs() ([]DiscoveredLog, error) {
	query := `
	SELECT id, file_path, app_name, file_size, last_modified, format, is_favorite, tag
	FROM discovered_logs
	ORDER BY last_modified DESC;
	`
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []DiscoveredLog
	for rows.Next() {
		var l DiscoveredLog
		var isFav int
		var modStr string
		if err := rows.Scan(&l.ID, &l.FilePath, &l.AppName, &l.FileSize, &modStr, &l.Format, &isFav, &l.Tag); err != nil {
			return nil, err
		}
		l.IsFavorite = (isFav == 1)
		t, parseErr := time.Parse(time.RFC3339, modStr)
		if parseErr == nil {
			l.LastModified = t
		}
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func (s *Store) RecordAIAccounting(entry AIAccountingEntry) error {
	query := `
	INSERT INTO ai_accounting (timestamp, provider, model, input_tokens, output_tokens, cost_usd, query_summary)
	VALUES (?, ?, ?, ?, ?, ?, ?);
	`
	_, err := s.db.Exec(query, entry.Timestamp.Format(time.RFC3339), entry.Provider, entry.Model, entry.InputTokens, entry.OutputTokens, entry.CostUSD, entry.QuerySummary)
	return err
}

func (s *Store) GetMonthlyAISpendUSD() (float64, error) {
	now := time.Now()
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location()).Format(time.RFC3339)

	query := `
	SELECT COALESCE(SUM(cost_usd), 0.0)
	FROM ai_accounting
	WHERE timestamp >= ?;
	`
	var total float64
	err := s.db.QueryRow(query, startOfMonth).Scan(&total)
	return total, err
}

func (s *Store) SetSetting(key, val string) error {
	query := `
	INSERT INTO settings (key, value)
	VALUES (?, ?)
	ON CONFLICT(key) DO UPDATE SET value = excluded.value;
	`
	_, err := s.db.Exec(query, key, val)
	return err
}

func (s *Store) GetSetting(key string) (string, error) {
	query := `SELECT value FROM settings WHERE key = ?;`
	var val string
	err := s.db.QueryRow(query, key).Scan(&val)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return val, err
}
