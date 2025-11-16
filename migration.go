package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Migration functions to import existing JSON data into SQLite

func MigrateJSONToDatabase(database *Database) error {
	fmt.Println("Starting migration of existing JSON data to SQLite database...")

	// Create repositories
	reviewRepo := NewSQLiteReviewStateRepository(database)
	dailyStatsRepo := NewSQLiteDailyStatsRepository(database)
	cardRepo := NewSQLiteCardRepository(database)

	// Migrate FSRS states
	if err := migrateFSRSStates(reviewRepo, cardRepo); err != nil {
		fmt.Printf("Warning: Failed to migrate FSRS states: %v\n", err)
	}

	// Migrate statistics
	if err := migrateStatistics(dailyStatsRepo); err != nil {
		fmt.Printf("Warning: Failed to migrate statistics: %v\n", err)
	}

	fmt.Println("Migration completed successfully!")
	return nil
}

func migrateFSRSStates(reviewRepo ReviewStateRepository, cardRepo CardRepository) error {
	stateFile := "./spaced_repetition_state.json"
	if _, err := os.Stat(stateFile); os.IsNotExist(err) {
		fmt.Printf("No FSRS state file found at %s, skipping FSRS migration\n", stateFile)
		return nil
	}

	data, err := os.ReadFile(stateFile)
	if err != nil {
		return fmt.Errorf("failed to read FSRS state file: %w", err)
	}

	var states map[string]*ReviewState
	if err := json.Unmarshal(data, &states); err != nil {
		return fmt.Errorf("failed to unmarshal FSRS states: %w", err)
	}

	fmt.Printf("Migrating %d FSRS review states...\n", len(states))

	// Get all cards to map file paths to database IDs
	allCards, err := cardRepo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get cards for mapping: %w", err)
	}

	// Create a map from file path + line number to card ID
	cardMapping := make(map[string]int64)
	for _, card := range allCards {
		key := fmt.Sprintf("%s:%d", card.SourceFile, card.SourceLine)
		cardMapping[key] = card.ID
	}

	migratedCount := 0
	for cardID, state := range states {
		// Find the corresponding database card ID
		dbCardID, exists := cardMapping[cardID]
		if !exists {
			fmt.Printf("Warning: Could not find database card for FSRS state: %s\n", cardID)
			continue
		}

		// Check if state already exists in database
		_, err := reviewRepo.GetByCardID(dbCardID)
		if err == nil {
			fmt.Printf("Review state already exists for card %d, skipping\n", dbCardID)
			continue
		}

		// Convert FSRS card to JSON
		fsrsCardJSON, err := FSRSCardToJSON(state.FSRSCard)
		if err != nil {
			fmt.Printf("Warning: Failed to convert FSRS card to JSON for %s: %v\n", cardID, err)
			continue
		}

		// Create database review state
		dbState := &DBReviewState{
			CardID:       dbCardID,
			FSRSCardData: fsrsCardJSON,
			LastReview:   state.LastReview,
			ReviewCount:  state.ReviewCount,
			DueDate:      state.FSRSCard.Due,
		}

		if err := reviewRepo.Create(dbState); err != nil {
			fmt.Printf("Warning: Failed to create review state for card %d: %v\n", dbCardID, err)
			continue
		}

		migratedCount++
	}

	fmt.Printf("Successfully migrated %d FSRS review states\n", migratedCount)
	return nil
}

func migrateStatistics(dailyStatsRepo DailyStatsRepository) error {
	statsFile := "./spaced_repetition_stats.json"
	if _, err := os.Stat(statsFile); os.IsNotExist(err) {
		fmt.Printf("No statistics file found at %s, skipping statistics migration\n", statsFile)
		return nil
	}

	data, err := os.ReadFile(statsFile)
	if err != nil {
		return fmt.Errorf("failed to read statistics file: %w", err)
	}

	// Structure matching the JSON format in statistics.go
	var statsData struct {
		DailyStats     map[string]*DailyStats `json:"daily_stats"`
		LearningStreak *LearningStreak        `json:"learning_streak"`
	}

	if err := json.Unmarshal(data, &statsData); err != nil {
		return fmt.Errorf("failed to unmarshal statistics: %w", err)
	}

	fmt.Printf("Migrating %d daily statistics records...\n", len(statsData.DailyStats))

	migratedCount := 0
	for date, stats := range statsData.DailyStats {
		// Check if stats already exist in database
		_, err := dailyStatsRepo.GetByDate(date)
		if err == nil {
			fmt.Printf("Daily stats already exist for date %s, skipping\n", date)
			continue
		}

		// Create database daily stats
		dbStats := &DBDailyStats{
			Date:         stats.Date,
			CardsReviewed: stats.CardsReviewed,
			SessionTime:  stats.SessionTime,
			SessionCount: stats.SessionCount,
			NewCards:     stats.NewCards,
			ReviewedCards: stats.ReviewedCards,
		}

		if err := dailyStatsRepo.Create(dbStats); err != nil {
			fmt.Printf("Warning: Failed to create daily stats for date %s: %v\n", date, err)
			continue
		}

		migratedCount++
	}

	fmt.Printf("Successfully migrated %d daily statistics records\n", migratedCount)
	return nil
}

// EnsureJSONFilesExist creates the state and stats JSON files if they don't exist
func EnsureJSONFilesExist() error {
	files := map[string]interface{}{
		"./spaced_repetition_state.json": map[string]interface{}{},
		"./spaced_repetition_stats.json": map[string]interface{}{
			"daily_stats":     map[string]interface{}{},
			"learning_streak": map[string]interface{}{},
		},
	}

	for filePath, defaultContent := range files {
		if _, err := os.Stat(filePath); err == nil {
			// File already exists, skip
			continue
		}

		data, err := json.MarshalIndent(defaultContent, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal JSON for %s: %w", filePath, err)
		}

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return fmt.Errorf("failed to create JSON file %s: %w", filePath, err)
		}

		fmt.Printf("Created JSON file: %s\n", filePath)
	}

	return nil
}

// Backup existing JSON files before migration
func BackupJSONFiles() error {
	timestamp := time.Now().Format("20060102_150405")

	filesToBackup := []string{
		"./spaced_repetition_state.json",
		"./spaced_repetition_stats.json",
	}

	for _, file := range filesToBackup {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			continue // File doesn't exist, skip
		}

		backupFile := file + ".backup_" + timestamp
		data, err := os.ReadFile(file)
		if err != nil {
			return fmt.Errorf("failed to read %s for backup: %w", file, err)
		}

		if err := os.WriteFile(backupFile, data, 0644); err != nil {
			return fmt.Errorf("failed to create backup %s: %w", backupFile, err)
		}

		fmt.Printf("Backed up %s to %s\n", file, backupFile)
	}

	return nil
}