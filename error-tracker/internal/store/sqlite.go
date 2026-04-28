package store

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"error-tracker/internal/model"
)

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	s := &SQLiteStore{db: db}
	if err := s.initTable(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *SQLiteStore) initTable() error {
	query := `
	CREATE TABLE IF NOT EXISTS error_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		app_id TEXT NOT NULL,
		type TEXT NOT NULL,
		message TEXT NOT NULL,
		stack TEXT DEFAULT '',
		url TEXT DEFAULT '',
		user_agent TEXT DEFAULT '',
		user_id TEXT DEFAULT '',
		meta TEXT DEFAULT '',
		timestamp INTEGER DEFAULT 0,
		created_at INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_app_id ON error_events(app_id);
	CREATE INDEX IF NOT EXISTS idx_created_at ON error_events(created_at);
	CREATE INDEX IF NOT EXISTS idx_type ON error_events(type);
	`
	_, err := s.db.Exec(query)
	return err
}

func (s *SQLiteStore) SaveEvent(event *model.ErrorEvent) error {
	event.CreatedAt = time.Now().UnixMilli()
	if event.Timestamp == 0 {
		event.Timestamp = event.CreatedAt
	}

	_, err := s.db.Exec(
		`INSERT INTO error_events (app_id, type, message, stack, url, user_agent, user_id, meta, timestamp, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.AppID, event.Type, event.Message, event.Stack, event.URL,
		event.UserAgent, event.UserID, event.Meta, event.Timestamp, event.CreatedAt,
	)
	return err
}

func (s *SQLiteStore) SaveEvents(events []*model.ErrorEvent) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(
		`INSERT INTO error_events (app_id, type, message, stack, url, user_agent, user_id, meta, timestamp, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	now := time.Now().UnixMilli()
	for _, e := range events {
		e.CreatedAt = now
		if e.Timestamp == 0 {
			e.Timestamp = now
		}
		_, err := stmt.Exec(e.AppID, e.Type, e.Message, e.Stack, e.URL,
			e.UserAgent, e.UserID, e.Meta, e.Timestamp, e.CreatedAt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *SQLiteStore) QueryEvents(appID, errorType string, page, size int) ([]*model.ErrorEvent, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	offset := (page - 1) * size

	var total int64
	var args []any

	countSQL := "SELECT COUNT(*) FROM error_events WHERE app_id = ?"
	querySQL := `SELECT id, app_id, type, message, stack, url, user_agent, user_id, meta, timestamp, created_at
		 FROM error_events WHERE app_id = ?`
	args = append(args, appID)

	if errorType != "" {
		countSQL += " AND type = ?"
		querySQL += " AND type = ?"
		args = append(args, errorType)
	}

	err := s.db.QueryRow(countSQL, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	querySQL += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	queryArgs := append(args, size, offset)

	rows, err := s.db.Query(querySQL, queryArgs...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var events []*model.ErrorEvent
	for rows.Next() {
		e := &model.ErrorEvent{}
		err := rows.Scan(&e.ID, &e.AppID, &e.Type, &e.Message, &e.Stack,
			&e.URL, &e.UserAgent, &e.UserID, &e.Meta, &e.Timestamp, &e.CreatedAt)
		if err != nil {
			return nil, 0, err
		}
		events = append(events, e)
	}

	return events, total, nil
}

func (s *SQLiteStore) GetTimeTrend(appID string, hours int) ([]*TimeBucket, error) {
	if hours < 1 {
		hours = 24
	}
	sinceMs := time.Now().Add(-time.Duration(hours) * time.Hour).UnixMilli()

	rows, err := s.db.Query(
		`SELECT strftime('%Y-%m-%d %H:00', created_at/1000, 'unixepoch', 'localtime') AS hour,
		        COUNT(*) AS count
		 FROM error_events
		 WHERE app_id = ? AND created_at >= ?
		 GROUP BY hour
		 ORDER BY hour ASC`,
		appID, sinceMs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var buckets []*TimeBucket
	for rows.Next() {
		b := &TimeBucket{}
		if err := rows.Scan(&b.Hour, &b.Count); err != nil {
			return nil, err
		}
		buckets = append(buckets, b)
	}
	return buckets, nil
}

func (s *SQLiteStore) GetEventStats(appID string) (*model.EventStats, error) {
	stats := &model.EventStats{
		AppID:  appID,
		ByType: make(map[string]int64),
	}

	err := s.db.QueryRow("SELECT COUNT(*) FROM error_events WHERE app_id = ?", appID).Scan(&stats.Total)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.Query(
		"SELECT type, COUNT(*) FROM error_events WHERE app_id = ? GROUP BY type", appID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var t string
		var count int64
		if err := rows.Scan(&t, &count); err != nil {
			return nil, err
		}
		stats.ByType[t] = count
	}

	return stats, nil
}

func (s *SQLiteStore) Close() error {
	return s.db.Close()
}
