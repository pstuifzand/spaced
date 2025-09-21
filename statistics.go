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
}

func NewStatisticsManager(statsFile string) *StatisticsManager {
	return &StatisticsManager{
		statsFile:  statsFile,
		dailyStats: make(map[string]*DailyStats),
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
}

func (sm *StatisticsManager) EndSession() {
	if sm.currentSession == nil {
		return
	}

	sm.currentSession.EndTime = time.Now()

	// Update daily stats
	today := time.Now().Format("2006-01-02")
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
	sessionDuration := int(sm.currentSession.EndTime.Sub(sm.currentSession.StartTime).Minutes())
	dailyStat.CardsReviewed += sm.currentSession.CardsReviewed
	dailyStat.SessionTime += sessionDuration
	dailyStat.SessionCount++
	dailyStat.NewCards += sm.currentSession.NewCards
	dailyStat.ReviewedCards += sm.currentSession.ReviewedCards

	// Update learning streak
	sm.updateLearningStreak(today)

	// Clear current session
	sm.currentSession = nil

	// Save stats
	sm.SaveStats()
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
			})
		}
	}

	return weekStats
}

func (sm *StatisticsManager) GetMonthlyStats() []DailyStats {
	today := time.Now()
	var monthStats []DailyStats

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
			})
		}
	}

	return monthStats
}

func (sm *StatisticsManager) GetAllTimeStats() (totalCards, totalTime, totalSessions int) {
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