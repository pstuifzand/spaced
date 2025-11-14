package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

type DailyStats struct {
	Date         string `json:"date"`          // YYYY-MM-DD format
	CardsReviewed int    `json:"cards_reviewed"`
	SessionTime  int    `json:"session_time"`  // minutes
	SessionCount int    `json:"session_count"`
	NewCards     int    `json:"new_cards"`
	ReviewedCards int   `json:"reviewed_cards"`
}

type SessionStats struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	CardsReviewed int      `json:"cards_reviewed"`
	NewCards     int       `json:"new_cards"`
	ReviewedCards int      `json:"reviewed_cards"`
}

type LearningStreak struct {
	CurrentStreak int    `json:"current_streak"`
	LongestStreak int    `json:"longest_streak"`
	LastStudyDate string `json:"last_study_date"` // YYYY-MM-DD
}

type StatisticsManager struct {
	statsFile       string
	dailyStats      map[string]*DailyStats // date -> stats
	currentSession  *SessionStats
	learningStreak  *LearningStreak

	// Database repositories
	sessionRepo       SessionRepository
	dailyStatsRepo    DailyStatsRepository
	useDatabase       bool
	currentSessionID  int64 // Track current database session ID
}

func NewStatisticsManager(statsFile string) *StatisticsManager {
	return &StatisticsManager{
		statsFile:   statsFile,
		dailyStats:  make(map[string]*DailyStats),
		useDatabase: false,
		learningStreak: &LearningStreak{
			CurrentStreak: 0,
			LongestStreak: 0,
			LastStudyDate: "",
		},
	}
}

func NewStatisticsManagerWithDatabase(sessionRepo SessionRepository, dailyStatsRepo DailyStatsRepository) *StatisticsManager {
	return &StatisticsManager{
		dailyStats:     make(map[string]*DailyStats),
		sessionRepo:    sessionRepo,
		dailyStatsRepo: dailyStatsRepo,
		useDatabase:    true,
		learningStreak: &LearningStreak{
			CurrentStreak: 0,
			LongestStreak: 0,
			LastStudyDate: "",
		},
	}
}

func (sm *StatisticsManager) LoadStats() error {
	if _, err := os.Stat(sm.statsFile); os.IsNotExist(err) {
		return nil // No stats file yet, start fresh
	}

	data, err := os.ReadFile(sm.statsFile)
	if err != nil {
		return fmt.Errorf("failed to read stats file: %w", err)
	}

	var statsData struct {
		DailyStats     map[string]*DailyStats `json:"daily_stats"`
		LearningStreak *LearningStreak        `json:"learning_streak"`
	}

	if err := json.Unmarshal(data, &statsData); err != nil {
		return fmt.Errorf("failed to parse stats file: %w", err)
	}

	sm.dailyStats = statsData.DailyStats
	if statsData.LearningStreak != nil {
		sm.learningStreak = statsData.LearningStreak
	}

	return nil
}

func (sm *StatisticsManager) SaveStats() error {
	dir := filepath.Dir(sm.statsFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	statsData := struct {
		DailyStats     map[string]*DailyStats `json:"daily_stats"`
		LearningStreak *LearningStreak        `json:"learning_streak"`
	}{
		DailyStats:     sm.dailyStats,
		LearningStreak: sm.learningStreak,
	}

	data, err := json.MarshalIndent(statsData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal stats: %w", err)
	}

	if err := os.WriteFile(sm.statsFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write stats file: %w", err)
	}

	return nil
}

func (sm *StatisticsManager) StartSession() {
	sm.currentSession = &SessionStats{
		StartTime:     time.Now(),
		CardsReviewed: 0,
		NewCards:      0,
		ReviewedCards: 0,
	}

	// If using database, create session record
	if sm.useDatabase && sm.sessionRepo != nil {
		dbSession := &DBSession{
			StartTime:     sm.currentSession.StartTime,
			CardsReviewed: 0,
			NewCards:      0,
			ReviewedCards: 0,
		}
		if err := sm.sessionRepo.Create(dbSession); err == nil {
			sm.currentSessionID = dbSession.ID
		}
	}
}

func (sm *StatisticsManager) HasActiveSession() bool {
	return sm.currentSession != nil
}

func (sm *StatisticsManager) EndSession() {
	if sm.currentSession == nil {
		return
	}

	sm.currentSession.EndTime = time.Now()
	sessionDuration := int(sm.currentSession.EndTime.Sub(sm.currentSession.StartTime).Minutes())
	today := time.Now().Format("2006-01-02")

	// Handle database or in-memory storage
	if sm.useDatabase && sm.dailyStatsRepo != nil {
		// Update session record in database
		if sm.sessionRepo != nil && sm.currentSessionID > 0 {
			dbSession := &DBSession{
				ID:            sm.currentSessionID,
				StartTime:     sm.currentSession.StartTime,
				EndTime:       sm.currentSession.EndTime,
				CardsReviewed: sm.currentSession.CardsReviewed,
				NewCards:      sm.currentSession.NewCards,
				ReviewedCards: sm.currentSession.ReviewedCards,
			}
			sm.sessionRepo.Update(dbSession)
		}

		// Get or create daily stats in database
		dbDailyStats, err := sm.dailyStatsRepo.GetByDate(today)
		if err != nil {
			// Create new daily stats
			dbDailyStats = &DBDailyStats{
				Date:         today,
				CardsReviewed: sm.currentSession.CardsReviewed,
				SessionTime:  sessionDuration,
				SessionCount: 1,
				NewCards:     sm.currentSession.NewCards,
				ReviewedCards: sm.currentSession.ReviewedCards,
			}
			sm.dailyStatsRepo.Create(dbDailyStats)
		} else {
			// Update existing daily stats
			dbDailyStats.CardsReviewed += sm.currentSession.CardsReviewed
			dbDailyStats.SessionTime += sessionDuration
			dbDailyStats.SessionCount++
			dbDailyStats.NewCards += sm.currentSession.NewCards
			dbDailyStats.ReviewedCards += sm.currentSession.ReviewedCards
			sm.dailyStatsRepo.Update(dbDailyStats)
		}
	} else {
		// In-memory storage (original behavior)
		dailyStat, exists := sm.dailyStats[today]
		if !exists {
			dailyStat = &DailyStats{
				Date:         today,
				CardsReviewed: 0,
				SessionTime:  0,
				SessionCount: 0,
				NewCards:     0,
				ReviewedCards: 0,
			}
			sm.dailyStats[today] = dailyStat
		}

		// Add session data to daily stats
		dailyStat.CardsReviewed += sm.currentSession.CardsReviewed
		dailyStat.SessionTime += sessionDuration
		dailyStat.SessionCount++
		dailyStat.NewCards += sm.currentSession.NewCards
		dailyStat.ReviewedCards += sm.currentSession.ReviewedCards
	}

	// Update learning streak
	sm.updateLearningStreak(today)

	// Clear current session
	sm.currentSession = nil
	sm.currentSessionID = 0

	// Save stats (for file-based mode)
	if !sm.useDatabase {
		sm.SaveStats()
	}
}

func (sm *StatisticsManager) RecordCardReview(isNewCard bool) {
	if sm.currentSession == nil {
		sm.StartSession()
	}

	sm.currentSession.CardsReviewed++
	if isNewCard {
		sm.currentSession.NewCards++
	} else {
		sm.currentSession.ReviewedCards++
	}

	// Immediately update database session if using database
	if sm.useDatabase && sm.sessionRepo != nil && sm.currentSessionID > 0 {
		dbSession := &DBSession{
			ID:            sm.currentSessionID,
			StartTime:     sm.currentSession.StartTime,
			EndTime:       time.Time{}, // Keep as zero until session ends
			CardsReviewed: sm.currentSession.CardsReviewed,
			NewCards:      sm.currentSession.NewCards,
			ReviewedCards: sm.currentSession.ReviewedCards,
		}
		sm.sessionRepo.Update(dbSession)
	}
}

func (sm *StatisticsManager) updateLearningStreak(today string) {
	if sm.learningStreak.LastStudyDate == "" {
		// First day studying
		sm.learningStreak.CurrentStreak = 1
		sm.learningStreak.LongestStreak = 1
		sm.learningStreak.LastStudyDate = today
		return
	}

	lastDate, err := time.Parse("2006-01-02", sm.learningStreak.LastStudyDate)
	if err != nil {
		// Reset on error
		sm.learningStreak.CurrentStreak = 1
		sm.learningStreak.LastStudyDate = today
		return
	}

	todayDate, _ := time.Parse("2006-01-02", today)
	daysDiff := int(todayDate.Sub(lastDate).Hours() / 24)

	switch daysDiff {
	case 0:
		// Same day, no change to streak
		return
	case 1:
		// Consecutive day, extend streak
		sm.learningStreak.CurrentStreak++
		if sm.learningStreak.CurrentStreak > sm.learningStreak.LongestStreak {
			sm.learningStreak.LongestStreak = sm.learningStreak.CurrentStreak
		}
	default:
		// Gap in studying, reset streak
		sm.learningStreak.CurrentStreak = 1
	}

	sm.learningStreak.LastStudyDate = today
}

func (sm *StatisticsManager) GetTodayStats() *DailyStats {
	today := time.Now().Format("2006-01-02")

	if sm.useDatabase && sm.dailyStatsRepo != nil {
		dbStats, err := sm.dailyStatsRepo.GetByDate(today)
		if err != nil {
			// No stats for today yet
			return &DailyStats{
				Date:         today,
				CardsReviewed: 0,
				SessionTime:  0,
				SessionCount: 0,
				NewCards:     0,
				ReviewedCards: 0,
			}
		}
		return &DailyStats{
			Date:         dbStats.Date,
			CardsReviewed: dbStats.CardsReviewed,
			SessionTime:  dbStats.SessionTime,
			SessionCount: dbStats.SessionCount,
			NewCards:     dbStats.NewCards,
			ReviewedCards: dbStats.ReviewedCards,
		}
	}

	// Fall back to in-memory stats
	stats, exists := sm.dailyStats[today]
	if !exists {
		return &DailyStats{
			Date:         today,
			CardsReviewed: 0,
			SessionTime:  0,
			SessionCount: 0,
			NewCards:     0,
			ReviewedCards: 0,
		}
	}
	return stats
}

func (sm *StatisticsManager) GetWeeklyStats() []DailyStats {
	today := time.Now()
	var weekStats []DailyStats

	if sm.useDatabase && sm.dailyStatsRepo != nil {
		// Query database for the last 7 days
		startDate := today.AddDate(0, 0, -6).Format("2006-01-02")
		endDate := today.Format("2006-01-02")

		dbStats, err := sm.dailyStatsRepo.GetDateRange(startDate, endDate)
		if err != nil {
			// Fall back to empty stats on error
			for i := 6; i >= 0; i-- {
				date := today.AddDate(0, 0, -i).Format("2006-01-02")
				weekStats = append(weekStats, DailyStats{
					Date:         date,
					CardsReviewed: 0,
					SessionTime:  0,
					SessionCount: 0,
					NewCards:     0,
					ReviewedCards: 0,
				})
			}
			return weekStats
		}

		// Convert DB stats to map for easy lookup
		dbStatsMap := make(map[string]*DBDailyStats)
		for _, stats := range dbStats {
			dbStatsMap[stats.Date] = stats
		}

		// Build week stats array
		for i := 6; i >= 0; i-- {
			date := today.AddDate(0, 0, -i).Format("2006-01-02")
			if dbStat, exists := dbStatsMap[date]; exists {
				weekStats = append(weekStats, DailyStats{
					Date:         dbStat.Date,
					CardsReviewed: dbStat.CardsReviewed,
					SessionTime:  dbStat.SessionTime,
					SessionCount: dbStat.SessionCount,
					NewCards:     dbStat.NewCards,
					ReviewedCards: dbStat.ReviewedCards,
				})
			} else {
				weekStats = append(weekStats, DailyStats{
					Date:         date,
					CardsReviewed: 0,
					SessionTime:  0,
					SessionCount: 0,
					NewCards:     0,
					ReviewedCards: 0,
				})
			}
		}
		return weekStats
	}

	// Fall back to in-memory stats
	for i := 6; i >= 0; i-- {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		if stats, exists := sm.dailyStats[date]; exists {
			weekStats = append(weekStats, *stats)
		} else {
			weekStats = append(weekStats, DailyStats{
				Date:         date,
				CardsReviewed: 0,
				SessionTime:  0,
				SessionCount: 0,
				NewCards:     0,
				ReviewedCards: 0,
			})
		}
	}

	return weekStats
}

func (sm *StatisticsManager) GetMonthlyStats() []DailyStats {
	today := time.Now()
	var monthStats []DailyStats

	if sm.useDatabase && sm.dailyStatsRepo != nil {
		// Query database for the last 30 days
		startDate := today.AddDate(0, 0, -29).Format("2006-01-02")
		endDate := today.Format("2006-01-02")

		dbStats, err := sm.dailyStatsRepo.GetDateRange(startDate, endDate)
		if err != nil {
			// Fall back to empty stats on error
			for i := 29; i >= 0; i-- {
				date := today.AddDate(0, 0, -i).Format("2006-01-02")
				monthStats = append(monthStats, DailyStats{
					Date:         date,
					CardsReviewed: 0,
					SessionTime:  0,
					SessionCount: 0,
					NewCards:     0,
					ReviewedCards: 0,
				})
			}
			return monthStats
		}

		// Convert DB stats to map for easy lookup
		dbStatsMap := make(map[string]*DBDailyStats)
		for _, stats := range dbStats {
			dbStatsMap[stats.Date] = stats
		}

		// Build month stats array
		for i := 29; i >= 0; i-- {
			date := today.AddDate(0, 0, -i).Format("2006-01-02")
			if dbStat, exists := dbStatsMap[date]; exists {
				monthStats = append(monthStats, DailyStats{
					Date:         dbStat.Date,
					CardsReviewed: dbStat.CardsReviewed,
					SessionTime:  dbStat.SessionTime,
					SessionCount: dbStat.SessionCount,
					NewCards:     dbStat.NewCards,
					ReviewedCards: dbStat.ReviewedCards,
				})
			} else {
				monthStats = append(monthStats, DailyStats{
					Date:         date,
					CardsReviewed: 0,
					SessionTime:  0,
					SessionCount: 0,
					NewCards:     0,
					ReviewedCards: 0,
				})
			}
		}
		return monthStats
	}

	// Fall back to in-memory stats
	for i := 29; i >= 0; i-- {
		date := today.AddDate(0, 0, -i).Format("2006-01-02")
		if stats, exists := sm.dailyStats[date]; exists {
			monthStats = append(monthStats, *stats)
		} else {
			monthStats = append(monthStats, DailyStats{
				Date:         date,
				CardsReviewed: 0,
				SessionTime:  0,
				SessionCount: 0,
				NewCards:     0,
				ReviewedCards: 0,
			})
		}
	}

	return monthStats
}

func (sm *StatisticsManager) GetAllTimeStats() (totalCards, totalTime, totalSessions int) {
	if sm.useDatabase && sm.dailyStatsRepo != nil {
		// Query all stats from database
		dbStats, err := sm.dailyStatsRepo.GetAll()
		if err != nil {
			return 0, 0, 0
		}

		for _, stats := range dbStats {
			totalCards += stats.CardsReviewed
			totalTime += stats.SessionTime
			totalSessions += stats.SessionCount
		}
		return
	}

	// Fall back to in-memory stats
	for _, stats := range sm.dailyStats {
		totalCards += stats.CardsReviewed
		totalTime += stats.SessionTime
		totalSessions += stats.SessionCount
	}
	return
}

func (sm *StatisticsManager) GetLearningStreak() *LearningStreak {
	return sm.learningStreak
}

func (sm *StatisticsManager) GetCurrentSessionDuration() time.Duration {
	if sm.currentSession == nil {
		return 0
	}
	return time.Since(sm.currentSession.StartTime)
}

func (sm *StatisticsManager) GetCurrentSessionStats() *SessionStats {
	return sm.currentSession
}

func (sm *StatisticsManager) CleanupOrphanedSessions() error {
	if !sm.useDatabase || sm.sessionRepo == nil {
		return nil // Only applicable for database mode
	}

	// First, end any unfinished sessions that have card activity
	if err := sm.endUnfinishedSessions(); err != nil {
		fmt.Printf("Warning: Failed to end unfinished sessions: %v\n", err)
	}

	// Then delete completely empty orphaned sessions
	deletedCount, err := sm.sessionRepo.DeleteOrphanedSessions()
	if err != nil {
		return fmt.Errorf("failed to delete orphaned sessions: %w", err)
	}

	if deletedCount > 0 {
		fmt.Printf("Cleaned up %d orphaned sessions with no activity\n", deletedCount)
	}

	return nil
}

func (sm *StatisticsManager) endUnfinishedSessions() error {
	// Get all sessions without end times
	sessions, err := sm.sessionRepo.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get sessions: %w", err)
	}

	endedCount := 0
	for _, session := range sessions {
		// If session has cards reviewed but no end time, estimate end time and close it
		if session.EndTime.IsZero() && session.CardsReviewed > 0 {
			// Estimate end time: assume 30 seconds per card reviewed
			estimatedDuration := time.Duration(session.CardsReviewed * 30) * time.Second
			estimatedEndTime := session.StartTime.Add(estimatedDuration)

			// Update session with estimated end time
			session.EndTime = estimatedEndTime
			if err := sm.sessionRepo.Update(session); err != nil {
				fmt.Printf("Warning: Failed to update session %d: %v\n", session.ID, err)
				continue
			}

			// Add to daily stats
			sm.aggregateSessionToDaily(session)
			endedCount++
		}
	}

	if endedCount > 0 {
		fmt.Printf("Ended %d unfinished sessions and added to daily statistics\n", endedCount)
	}

	return nil
}

func (sm *StatisticsManager) aggregateSessionToDaily(session *DBSession) {
	if !sm.useDatabase || sm.dailyStatsRepo == nil {
		return
	}

	today := session.StartTime.Format("2006-01-02")
	sessionDuration := int(session.EndTime.Sub(session.StartTime).Minutes())

	// Get or create daily stats
	dbDailyStats, err := sm.dailyStatsRepo.GetByDate(today)
	if err != nil {
		// Create new daily stats
		dbDailyStats = &DBDailyStats{
			Date:         today,
			CardsReviewed: session.CardsReviewed,
			SessionTime:  sessionDuration,
			SessionCount: 1,
			NewCards:     session.NewCards,
			ReviewedCards: session.ReviewedCards,
		}
		sm.dailyStatsRepo.Create(dbDailyStats)
	} else {
		// Update existing daily stats
		dbDailyStats.CardsReviewed += session.CardsReviewed
		dbDailyStats.SessionTime += sessionDuration
		dbDailyStats.SessionCount++
		dbDailyStats.NewCards += session.NewCards
		dbDailyStats.ReviewedCards += session.ReviewedCards
		sm.dailyStatsRepo.Update(dbDailyStats)
	}
}

func (sm *StatisticsManager) ExportToCSV(filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create CSV file: %w", err)
	}
	defer file.Close()

	// Write CSV header
	_, err = file.WriteString("Date,Cards Reviewed,Session Time (min),Session Count,New Cards,Reviewed Cards\n")
	if err != nil {
		return err
	}

	// Sort dates
	var dates []string
	for date := range sm.dailyStats {
		dates = append(dates, date)
	}
	sort.Strings(dates)

	// Write data
	for _, date := range dates {
		stats := sm.dailyStats[date]
		line := fmt.Sprintf("%s,%d,%d,%d,%d,%d\n",
			stats.Date, stats.CardsReviewed, stats.SessionTime,
			stats.SessionCount, stats.NewCards, stats.ReviewedCards)
		_, err = file.WriteString(line)
		if err != nil {
			return err
		}
	}

	return nil
}