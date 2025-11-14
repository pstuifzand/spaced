package main

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type Database struct {
	db *sql.DB
}

func NewDatabase(dbPath string) (*Database, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	database := &Database{db: db}
	if err := database.createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	// Run migrations for existing databases
	if err := database.migrateSchema(); err != nil {
		return nil, fmt.Errorf("failed to migrate schema: %w", err)
	}

	return database, nil
}

func (d *Database) Close() error {
	return d.db.Close()
}

func (d *Database) migrateSchema() error {
	// Check if new columns exist, add them if they don't
	migrations := []string{
		`ALTER TABLE cards ADD COLUMN source_context TEXT`,
		`ALTER TABLE cards ADD COLUMN prompt_type TEXT DEFAULT 'factual'`,
		`ALTER TABLE cards ADD COLUMN tags TEXT`,
	}

	for _, migration := range migrations {
		// Try to execute migration; it will fail if column already exists (which is fine)
		d.db.Exec(migration)
	}

	return nil
}

func (d *Database) createTables() error {
	schemas := []string{
		`CREATE TABLE IF NOT EXISTS cards (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			question TEXT NOT NULL,
			answer TEXT NOT NULL,
			source_file TEXT,
			source_line INTEGER,
			source_context TEXT,
			prompt_type TEXT DEFAULT 'factual',
			tags TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE TABLE IF NOT EXISTS review_states (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			card_id INTEGER NOT NULL,
			fsrs_card_data TEXT NOT NULL,
			last_review DATETIME,
			review_count INTEGER DEFAULT 0,
			due_date DATETIME,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (card_id) REFERENCES cards(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			start_time DATETIME NOT NULL,
			end_time DATETIME,
			cards_reviewed INTEGER DEFAULT 0,
			new_cards INTEGER DEFAULT 0,
			reviewed_cards INTEGER DEFAULT 0
		)`,
		`CREATE TABLE IF NOT EXISTS daily_stats (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			date DATE NOT NULL UNIQUE,
			cards_reviewed INTEGER DEFAULT 0,
			session_time INTEGER DEFAULT 0,
			session_count INTEGER DEFAULT 0,
			new_cards INTEGER DEFAULT 0,
			reviewed_cards INTEGER DEFAULT 0
		)`,
	}

	for _, schema := range schemas {
		if _, err := d.db.Exec(schema); err != nil {
			return fmt.Errorf("failed to create table: %w", err)
		}
	}

	// Create indexes for better performance
	indexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_cards_question ON cards(question)`,
		`CREATE INDEX IF NOT EXISTS idx_review_states_card_id ON review_states(card_id)`,
		`CREATE INDEX IF NOT EXISTS idx_review_states_due_date ON review_states(due_date)`,
		`CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date)`,
	}

	for _, index := range indexes {
		if _, err := d.db.Exec(index); err != nil {
			return fmt.Errorf("failed to create index: %w", err)
		}
	}

	return nil
}

// Database card structure
type DBCard struct {
	ID            int64     `db:"id"`
	Question      string    `db:"question"`
	Answer        string    `db:"answer"`
	SourceFile    string    `db:"source_file"`
	SourceLine    int       `db:"source_line"`
	SourceContext string    `db:"source_context"`
	PromptType    string    `db:"prompt_type"`
	Tags          string    `db:"tags"`
	CreatedAt     time.Time `db:"created_at"`
	UpdatedAt     time.Time `db:"updated_at"`
}

// Database review state structure
type DBReviewState struct {
	ID           int64     `db:"id"`
	CardID       int64     `db:"card_id"`
	FSRSCardData string    `db:"fsrs_card_data"`
	LastReview   time.Time `db:"last_review"`
	ReviewCount  int       `db:"review_count"`
	DueDate      time.Time `db:"due_date"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Database session structure
type DBSession struct {
	ID            int64     `db:"id"`
	StartTime     time.Time `db:"start_time"`
	EndTime       time.Time `db:"end_time"`
	CardsReviewed int       `db:"cards_reviewed"`
	NewCards      int       `db:"new_cards"`
	ReviewedCards int       `db:"reviewed_cards"`
}

// Database daily stats structure
type DBDailyStats struct {
	ID            int64  `db:"id"`
	Date          string `db:"date"`
	CardsReviewed int    `db:"cards_reviewed"`
	SessionTime   int    `db:"session_time"`
	SessionCount  int    `db:"session_count"`
	NewCards      int    `db:"new_cards"`
	ReviewedCards int    `db:"reviewed_cards"`
}