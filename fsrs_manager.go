package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/open-spaced-repetition/go-fsrs/v3"
)

type ReviewState struct {
	CardID      string     `json:"card_id"`
	FSRSCard    fsrs.Card  `json:"fsrs_card"`
	LastReview  time.Time  `json:"last_review"`
	ReviewCount int        `json:"review_count"`
}

type FSRSManager struct {
	fsrs         *fsrs.FSRS
	states       map[string]*ReviewState
	stateFile    string
	reviewRepo   ReviewStateRepository
	useDatabase  bool
}

func NewFSRSManager(stateFile string) *FSRSManager {
	return &FSRSManager{
		fsrs:        fsrs.NewFSRS(fsrs.DefaultParam()),
		states:      make(map[string]*ReviewState),
		stateFile:   stateFile,
		useDatabase: false,
	}
}

func NewFSRSManagerWithDatabase(reviewRepo ReviewStateRepository) *FSRSManager {
	return &FSRSManager{
		fsrs:        fsrs.NewFSRS(fsrs.DefaultParam()),
		states:      make(map[string]*ReviewState),
		reviewRepo:  reviewRepo,
		useDatabase: true,
	}
}

func (fm *FSRSManager) LoadState() error {
	if _, err := os.Stat(fm.stateFile); os.IsNotExist(err) {
		return nil
	}

	data, err := os.ReadFile(fm.stateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var states map[string]*ReviewState
	if err := json.Unmarshal(data, &states); err != nil {
		return fmt.Errorf("failed to parse state file: %w", err)
	}

	fm.states = states
	return nil
}

func (fm *FSRSManager) SaveState() error {
	dir := filepath.Dir(fm.stateFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	data, err := json.MarshalIndent(fm.states, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(fm.stateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

func (fm *FSRSManager) getCardID(card Card) string {
	return fmt.Sprintf("%s:%d", card.FilePath, card.LineNum)
}

func (fm *FSRSManager) getCardDBID(card Card) int64 {
	// For database mode, we need to find the card ID based on question/answer
	// This is a temporary solution - ideally we'd pass the DB ID with the card
	return int64(card.LineNum) // Placeholder - will be improved
}

func (fm *FSRSManager) GetCardState(card Card) *ReviewState {
	if fm.useDatabase && fm.reviewRepo != nil && card.ID > 0 {
		// Try to get from database first
		dbState, err := fm.reviewRepo.GetByCardID(card.ID)
		if err == nil {
			// Convert DB state to ReviewState
			fsrsCard, err := JSONToFSRSCard(dbState.FSRSCardData)
			if err != nil {
				// If JSON parsing fails, create new card
				fsrsCard = fsrs.NewCard()
			}
			return &ReviewState{
				CardID:      fm.getCardID(card),
				FSRSCard:    fsrsCard,
				LastReview:  dbState.LastReview,
				ReviewCount: dbState.ReviewCount,
			}
		}

		// If not found in database, create new state
		newState := &ReviewState{
			CardID:      fm.getCardID(card),
			FSRSCard:    fsrs.NewCard(),
			LastReview:  time.Time{},
			ReviewCount: 0,
		}

		// Save to database
		fsrsCardJSON, _ := FSRSCardToJSON(newState.FSRSCard)
		dbState = &DBReviewState{
			CardID:       card.ID,
			FSRSCardData: fsrsCardJSON,
			LastReview:   newState.LastReview,
			ReviewCount:  newState.ReviewCount,
			DueDate:      newState.FSRSCard.Due,
		}
		fm.reviewRepo.Create(dbState)

		return newState
	}

	// Fall back to memory-based approach
	cardID := fm.getCardID(card)
	state, exists := fm.states[cardID]
	if !exists {
		state = &ReviewState{
			CardID:      cardID,
			FSRSCard:    fsrs.NewCard(),
			LastReview:  time.Time{},
			ReviewCount: 0,
		}
		fm.states[cardID] = state
	}
	return state
}

func (fm *FSRSManager) IsCardDue(card Card) bool {
	state := fm.GetCardState(card)

	if state.ReviewCount == 0 {
		return true
	}

	return time.Now().After(state.FSRSCard.Due)
}

func (fm *FSRSManager) ReviewCard(card Card, rating fsrs.Rating) error {
	state := fm.GetCardState(card)
	now := time.Now()

	schedulingInfo := fm.fsrs.Next(state.FSRSCard, now, rating)

	state.FSRSCard = schedulingInfo.Card
	state.LastReview = now
	state.ReviewCount++

	// Save to database if using database mode
	if fm.useDatabase && fm.reviewRepo != nil && card.ID > 0 {
		fsrsCardJSON, err := FSRSCardToJSON(state.FSRSCard)
		if err != nil {
			return fmt.Errorf("failed to convert FSRS card to JSON: %w", err)
		}

		dbState := &DBReviewState{
			CardID:       card.ID,
			FSRSCardData: fsrsCardJSON,
			LastReview:   state.LastReview,
			ReviewCount:  state.ReviewCount,
			DueDate:      state.FSRSCard.Due,
		}

		// Try to update existing state
		existing, err := fm.reviewRepo.GetByCardID(card.ID)
		if err != nil {
			// Create new state
			return fm.reviewRepo.Create(dbState)
		} else {
			// Update existing state
			dbState.ID = existing.ID
			return fm.reviewRepo.Update(dbState)
		}
	}

	// Fall back to file-based saving
	return fm.SaveState()
}

func (fm *FSRSManager) GetDueCards(cards []Card) []Card {
	var dueCards []Card
	for _, card := range cards {
		if fm.IsCardDue(card) {
			dueCards = append(dueCards, card)
		}
	}
	return dueCards
}

func (fm *FSRSManager) GetStats(cards []Card) (total, due, reviewed int) {
	total = len(cards)
	for _, card := range cards {
		state := fm.GetCardState(card)
		if fm.IsCardDue(card) {
			due++
		}
		if state.ReviewCount > 0 {
			reviewed++
		}
	}
	return
}

func (fm *FSRSManager) DeleteCardState(cardID int64) error {
	// Create card key for in-memory lookup
	cardKey := fmt.Sprintf("%d", cardID)

	// Remove from in-memory cache
	delete(fm.states, cardKey)

	// Remove from database if using database mode
	if fm.useDatabase && fm.reviewRepo != nil {
		return fm.reviewRepo.Delete(cardID)
	}

	return nil
}