package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/open-spaced-repetition/go-fsrs/v3"
)

// Repository interfaces
type CardRepository interface {
	Create(card *DBCard) error
	GetByID(id int64) (*DBCard, error)
	GetAll() ([]*DBCard, error)
	Update(card *DBCard) error
	Delete(id int64) error
	ImportFromText(question, answer, sourceFile string, sourceLine int) (*DBCard, error)
	CardExists(question, answer string) (bool, error)
}

type ReviewStateRepository interface {
	Create(state *DBReviewState) error
	GetByCardID(cardID int64) (*DBReviewState, error)
	Update(state *DBReviewState) error
	Delete(cardID int64) error
	GetDueCards() ([]*DBReviewState, error)
}

type SessionRepository interface {
	Create(session *DBSession) error
	GetByID(id int64) (*DBSession, error)
	Update(session *DBSession) error
	GetAll() ([]*DBSession, error)
	Delete(id int64) error
	DeleteOrphanedSessions() (int, error)
}

type DailyStatsRepository interface {
	Create(stats *DBDailyStats) error
	GetByDate(date string) (*DBDailyStats, error)
	Update(stats *DBDailyStats) error
	GetDateRange(startDate, endDate string) ([]*DBDailyStats, error)
	GetAll() ([]*DBDailyStats, error)
}

// SQLite implementations
type SQLiteCardRepository struct {
	db *Database
}

func NewSQLiteCardRepository(db *Database) *SQLiteCardRepository {
	return &SQLiteCardRepository{db: db}
}

func (r *SQLiteCardRepository) Create(card *DBCard) error {
	query := `INSERT INTO cards (question, answer, source_file, source_line, source_context, prompt_type, tags, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	card.CreatedAt = now
	card.UpdatedAt = now

	// Set default prompt type if not provided
	if card.PromptType == "" {
		card.PromptType = "factual"
	}

	result, err := r.db.db.Exec(query, card.Question, card.Answer, card.SourceFile, card.SourceLine,
								card.SourceContext, card.PromptType, card.Tags, now, now)
	if err != nil {
		return fmt.Errorf("failed to create card: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	card.ID = id
	return nil
}

func (r *SQLiteCardRepository) GetByID(id int64) (*DBCard, error) {
	query := `SELECT id, question, answer, source_file, source_line, source_context, prompt_type, tags, created_at, updated_at
			  FROM cards WHERE id = ?`

	row := r.db.db.QueryRow(query, id)

	card := &DBCard{}
	err := row.Scan(&card.ID, &card.Question, &card.Answer, &card.SourceFile,
					&card.SourceLine, &card.SourceContext, &card.PromptType, &card.Tags,
					&card.CreatedAt, &card.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get card: %w", err)
	}

	return card, nil
}

func (r *SQLiteCardRepository) GetAll() ([]*DBCard, error) {
	query := `SELECT id, question, answer, source_file, source_line, source_context, prompt_type, tags, created_at, updated_at
			  FROM cards ORDER BY created_at ASC`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query cards: %w", err)
	}
	defer rows.Close()

	var cards []*DBCard
	for rows.Next() {
		card := &DBCard{}
		err := rows.Scan(&card.ID, &card.Question, &card.Answer, &card.SourceFile,
						&card.SourceLine, &card.SourceContext, &card.PromptType, &card.Tags,
						&card.CreatedAt, &card.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan card: %w", err)
		}
		cards = append(cards, card)
	}

	return cards, nil
}

func (r *SQLiteCardRepository) Update(card *DBCard) error {
	query := `UPDATE cards SET question = ?, answer = ?, source_file = ?,
			  source_line = ?, source_context = ?, prompt_type = ?, tags = ?, updated_at = ? WHERE id = ?`

	card.UpdatedAt = time.Now()

	_, err := r.db.db.Exec(query, card.Question, card.Answer, card.SourceFile,
						   card.SourceLine, card.SourceContext, card.PromptType, card.Tags,
						   card.UpdatedAt, card.ID)
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}

	return nil
}

func (r *SQLiteCardRepository) Delete(id int64) error {
	query := `DELETE FROM cards WHERE id = ?`

	_, err := r.db.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}

	return nil
}

func (r *SQLiteCardRepository) ImportFromText(question, answer, sourceFile string, sourceLine int) (*DBCard, error) {
	card := &DBCard{
		Question:      question,
		Answer:        answer,
		SourceFile:    sourceFile,
		SourceLine:    sourceLine,
		SourceContext: "", // Will be empty for imported text files
		PromptType:    "factual",
		Tags:          "",
	}

	err := r.Create(card)
	if err != nil {
		return nil, fmt.Errorf("failed to import card: %w", err)
	}

	return card, nil
}

func (r *SQLiteCardRepository) CardExists(question, answer string) (bool, error) {
	query := `SELECT COUNT(*) FROM cards WHERE question = ? AND answer = ?`

	var count int
	err := r.db.db.QueryRow(query, question, answer).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if card exists: %w", err)
	}

	return count > 0, nil
}

// SQLite Review State Repository
type SQLiteReviewStateRepository struct {
	db *Database
}

func NewSQLiteReviewStateRepository(db *Database) *SQLiteReviewStateRepository {
	return &SQLiteReviewStateRepository{db: db}
}

func (r *SQLiteReviewStateRepository) Create(state *DBReviewState) error {
	query := `INSERT INTO review_states (card_id, fsrs_card_data, last_review, review_count, due_date, created_at, updated_at)
			  VALUES (?, ?, ?, ?, ?, ?, ?)`

	now := time.Now()
	state.CreatedAt = now
	state.UpdatedAt = now

	result, err := r.db.db.Exec(query, state.CardID, state.FSRSCardData, state.LastReview,
								state.ReviewCount, state.DueDate, now, now)
	if err != nil {
		return fmt.Errorf("failed to create review state: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	state.ID = id
	return nil
}

func (r *SQLiteReviewStateRepository) GetByCardID(cardID int64) (*DBReviewState, error) {
	query := `SELECT id, card_id, fsrs_card_data, last_review, review_count, due_date, created_at, updated_at
			  FROM review_states WHERE card_id = ?`

	row := r.db.db.QueryRow(query, cardID)

	state := &DBReviewState{}
	err := row.Scan(&state.ID, &state.CardID, &state.FSRSCardData, &state.LastReview,
					&state.ReviewCount, &state.DueDate, &state.CreatedAt, &state.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to get review state: %w", err)
	}

	return state, nil
}

func (r *SQLiteReviewStateRepository) Update(state *DBReviewState) error {
	query := `UPDATE review_states SET fsrs_card_data = ?, last_review = ?,
			  review_count = ?, due_date = ?, updated_at = ? WHERE card_id = ?`

	state.UpdatedAt = time.Now()

	_, err := r.db.db.Exec(query, state.FSRSCardData, state.LastReview,
						   state.ReviewCount, state.DueDate, state.UpdatedAt, state.CardID)
	if err != nil {
		return fmt.Errorf("failed to update review state: %w", err)
	}

	return nil
}

func (r *SQLiteReviewStateRepository) Delete(cardID int64) error {
	query := `DELETE FROM review_states WHERE card_id = ?`

	_, err := r.db.db.Exec(query, cardID)
	if err != nil {
		return fmt.Errorf("failed to delete review state: %w", err)
	}

	return nil
}

func (r *SQLiteReviewStateRepository) GetDueCards() ([]*DBReviewState, error) {
	query := `SELECT id, card_id, fsrs_card_data, last_review, review_count, due_date, created_at, updated_at
			  FROM review_states WHERE due_date <= ? ORDER BY due_date ASC`

	now := time.Now()
	rows, err := r.db.db.Query(query, now)
	if err != nil {
		return nil, fmt.Errorf("failed to query due cards: %w", err)
	}
	defer rows.Close()

	var states []*DBReviewState
	for rows.Next() {
		state := &DBReviewState{}
		err := rows.Scan(&state.ID, &state.CardID, &state.FSRSCardData, &state.LastReview,
						&state.ReviewCount, &state.DueDate, &state.CreatedAt, &state.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan review state: %w", err)
		}
		states = append(states, state)
	}

	return states, nil
}

// Utility functions for converting between FSRS cards and JSON
func FSRSCardToJSON(card fsrs.Card) (string, error) {
	data, err := json.Marshal(card)
	if err != nil {
		return "", fmt.Errorf("failed to marshal FSRS card: %w", err)
	}
	return string(data), nil
}

func JSONToFSRSCard(data string) (fsrs.Card, error) {
	var card fsrs.Card
	err := json.Unmarshal([]byte(data), &card)
	if err != nil {
		return card, fmt.Errorf("failed to unmarshal FSRS card: %w", err)
	}
	return card, nil
}

// SQLite Session Repository
type SQLiteSessionRepository struct {
	db *Database
}

func NewSQLiteSessionRepository(db *Database) *SQLiteSessionRepository {
	return &SQLiteSessionRepository{db: db}
}

func (r *SQLiteSessionRepository) Create(session *DBSession) error {
	query := `INSERT INTO sessions (start_time, end_time, cards_reviewed, new_cards, reviewed_cards)
			  VALUES (?, ?, ?, ?, ?)`

	result, err := r.db.db.Exec(query, session.StartTime, session.EndTime,
								session.CardsReviewed, session.NewCards, session.ReviewedCards)
	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	session.ID = id
	return nil
}

func (r *SQLiteSessionRepository) GetByID(id int64) (*DBSession, error) {
	query := `SELECT id, start_time, end_time, cards_reviewed, new_cards, reviewed_cards
			  FROM sessions WHERE id = ?`

	row := r.db.db.QueryRow(query, id)

	session := &DBSession{}
	err := row.Scan(&session.ID, &session.StartTime, &session.EndTime,
					&session.CardsReviewed, &session.NewCards, &session.ReviewedCards)
	if err != nil {
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

func (r *SQLiteSessionRepository) Update(session *DBSession) error {
	query := `UPDATE sessions SET start_time = ?, end_time = ?, cards_reviewed = ?,
			  new_cards = ?, reviewed_cards = ? WHERE id = ?`

	_, err := r.db.db.Exec(query, session.StartTime, session.EndTime,
						   session.CardsReviewed, session.NewCards, session.ReviewedCards, session.ID)
	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	return nil
}

func (r *SQLiteSessionRepository) GetAll() ([]*DBSession, error) {
	query := `SELECT id, start_time, end_time, cards_reviewed, new_cards, reviewed_cards
			  FROM sessions ORDER BY start_time DESC`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*DBSession
	for rows.Next() {
		session := &DBSession{}
		err := rows.Scan(&session.ID, &session.StartTime, &session.EndTime,
						&session.CardsReviewed, &session.NewCards, &session.ReviewedCards)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

func (r *SQLiteSessionRepository) Delete(id int64) error {
	query := `DELETE FROM sessions WHERE id = ?`

	_, err := r.db.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete session: %w", err)
	}

	return nil
}

func (r *SQLiteSessionRepository) DeleteOrphanedSessions() (int, error) {
	// Delete sessions that have no end time and no cards reviewed (orphaned sessions)
	query := `DELETE FROM sessions WHERE (end_time IS NULL OR end_time = '0001-01-01 00:00:00+00:00') AND cards_reviewed = 0`

	result, err := r.db.db.Exec(query)
	if err != nil {
		return 0, fmt.Errorf("failed to delete orphaned sessions: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return int(rowsAffected), nil
}

// SQLite Daily Stats Repository
type SQLiteDailyStatsRepository struct {
	db *Database
}

func NewSQLiteDailyStatsRepository(db *Database) *SQLiteDailyStatsRepository {
	return &SQLiteDailyStatsRepository{db: db}
}

func (r *SQLiteDailyStatsRepository) Create(stats *DBDailyStats) error {
	query := `INSERT INTO daily_stats (date, cards_reviewed, session_time, session_count, new_cards, reviewed_cards)
			  VALUES (?, ?, ?, ?, ?, ?)`

	result, err := r.db.db.Exec(query, stats.Date, stats.CardsReviewed,
								stats.SessionTime, stats.SessionCount, stats.NewCards, stats.ReviewedCards)
	if err != nil {
		return fmt.Errorf("failed to create daily stats: %w", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		return fmt.Errorf("failed to get last insert id: %w", err)
	}

	stats.ID = id
	return nil
}

func (r *SQLiteDailyStatsRepository) GetByDate(date string) (*DBDailyStats, error) {
	query := `SELECT id, date, cards_reviewed, session_time, session_count, new_cards, reviewed_cards
			  FROM daily_stats WHERE date = ?`

	row := r.db.db.QueryRow(query, date)

	stats := &DBDailyStats{}
	err := row.Scan(&stats.ID, &stats.Date, &stats.CardsReviewed,
					&stats.SessionTime, &stats.SessionCount, &stats.NewCards, &stats.ReviewedCards)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily stats: %w", err)
	}

	return stats, nil
}

func (r *SQLiteDailyStatsRepository) Update(stats *DBDailyStats) error {
	query := `UPDATE daily_stats SET cards_reviewed = ?, session_time = ?,
			  session_count = ?, new_cards = ?, reviewed_cards = ? WHERE date = ?`

	_, err := r.db.db.Exec(query, stats.CardsReviewed, stats.SessionTime,
						   stats.SessionCount, stats.NewCards, stats.ReviewedCards, stats.Date)
	if err != nil {
		return fmt.Errorf("failed to update daily stats: %w", err)
	}

	return nil
}

func (r *SQLiteDailyStatsRepository) GetDateRange(startDate, endDate string) ([]*DBDailyStats, error) {
	query := `SELECT id, date, cards_reviewed, session_time, session_count, new_cards, reviewed_cards
			  FROM daily_stats WHERE date BETWEEN ? AND ? ORDER BY date DESC`

	rows, err := r.db.db.Query(query, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*DBDailyStats
	for rows.Next() {
		stat := &DBDailyStats{}
		err := rows.Scan(&stat.ID, &stat.Date, &stat.CardsReviewed,
						&stat.SessionTime, &stat.SessionCount, &stat.NewCards, &stat.ReviewedCards)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily stats: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}

func (r *SQLiteDailyStatsRepository) GetAll() ([]*DBDailyStats, error) {
	query := `SELECT id, date, cards_reviewed, session_time, session_count, new_cards, reviewed_cards
			  FROM daily_stats ORDER BY date DESC`

	rows, err := r.db.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query daily stats: %w", err)
	}
	defer rows.Close()

	var stats []*DBDailyStats
	for rows.Next() {
		stat := &DBDailyStats{}
		err := rows.Scan(&stat.ID, &stat.Date, &stat.CardsReviewed,
						&stat.SessionTime, &stat.SessionCount, &stat.NewCards, &stat.ReviewedCards)
		if err != nil {
			return nil, fmt.Errorf("failed to scan daily stats: %w", err)
		}
		stats = append(stats, stat)
	}

	return stats, nil
}