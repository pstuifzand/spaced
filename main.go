package main

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/open-spaced-repetition/go-fsrs/v3"
)

type SpacedRepetitionApp struct {
	app        fyne.App
	window     fyne.Window
	parser     *CardParser
	fsrsManager *FSRSManager
	statsManager *StatisticsManager

	currentCard    *Card
	currentIndex   int
	dueCards      []Card

	questionLabel  *widget.Label
	answerLabel    *widget.Label
	showAnswerBtn  *widget.Button
	ratingContainer *fyne.Container
	statsLabel     *widget.Label

	showingAnswer  bool
	sessionStarted bool
}

func NewSpacedRepetitionApp() *SpacedRepetitionApp {
	myApp := app.New()
	myApp.SetIcon(nil)
	myApp.Settings().SetTheme(&SpacedRepetitionTheme{})

	window := myApp.NewWindow("Spaced Repetition - Learn Efficiently")
	window.Resize(fyne.NewSize(900, 700))
	window.CenterOnScreen()

	sra := &SpacedRepetitionApp{
		app:          myApp,
		window:       window,
		parser:       NewCardParser(),
		fsrsManager:  NewFSRSManager("./spaced_repetition_state.json"),
		statsManager: NewStatisticsManager("./spaced_repetition_stats.json"),
		currentIndex: -1,
		sessionStarted: false,
	}

	// Setup menu bar
	sra.setupMenuBar()

	return sra
}

func (sra *SpacedRepetitionApp) setupMenuBar() {
	// Create File menu
	openCards := fyne.NewMenuItem("Open Cards...", func() {
		sra.loadCards()
	})

	addCard := fyne.NewMenuItem("Add New Card...", func() {
		sra.showAddCardDialog()
	})

	exportStats := fyne.NewMenuItem("Export Statistics...", func() {
		sra.exportStatistics()
	})

	// Create menu items
	fileMenu := fyne.NewMenu("File",
		openCards,
		fyne.NewMenuItemSeparator(),
		addCard,
		fyne.NewMenuItemSeparator(),
		exportStats,
	)

	// Create Statistics menu
	viewStats := fyne.NewMenuItem("View Statistics", func() {
		sra.showStatistics()
	})

	resetStats := fyne.NewMenuItem("Reset Statistics", func() {
		sra.resetStatistics()
	})

	statsMenu := fyne.NewMenu("Statistics",
		viewStats,
		fyne.NewMenuItemSeparator(),
		resetStats,
	)

	// Create Help menu
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			dialog.ShowInformation("About",
				"Spaced Repetition v1.0\n\nAn efficient learning tool using the FSRS algorithm.\n\nLoad cards in 'question>>answer' format and study efficiently!\n\n⌨️ Keyboard Shortcuts:\n• S = Show Answer\n• 1 = Again (red)\n• 2 = Hard (orange)\n• 3 = Good (green)\n• 4 = Easy (blue)\n• N = Add New Card\n\n📝 Add Card Dialog:\n• Tab = Navigate fields\n• Enter = Next field/Submit\n• Escape = Cancel",
				sra.window)
		}),
	)

	// Create main menu
	mainMenu := fyne.NewMainMenu(fileMenu, statsMenu, helpMenu)
	sra.window.SetMainMenu(mainMenu)
}

func (sra *SpacedRepetitionApp) setupUI() {
	// Question label - large and prominent
	sra.questionLabel = widget.NewLabelWithStyle("Welcome to Spaced Repetition!",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	sra.questionLabel.Wrapping = fyne.TextWrapWord

	// Answer label - clear but secondary, starts with placeholder text
	sra.answerLabel = widget.NewLabelWithStyle("",
		fyne.TextAlignCenter, fyne.TextStyle{})
	sra.answerLabel.Wrapping = fyne.TextWrapWord

	// Show answer button - prominent
	sra.showAnswerBtn = widget.NewButton("👁️ Show Answer (S)", sra.showAnswer)
	sra.showAnswerBtn.Importance = widget.HighImportance
	sra.showAnswerBtn.Hide()

	// Color-coded rating buttons with icons and keyboard shortcuts
	againBtn := widget.NewButtonWithIcon("❌ Again (1)", nil, func() {
		sra.rateCard(fsrs.Again)
	})
	againBtn.Importance = widget.DangerImportance

	hardBtn := widget.NewButtonWithIcon("⚠️ Hard (2)", nil, func() {
		sra.rateCard(fsrs.Hard)
	})
	hardBtn.Importance = widget.MediumImportance

	goodBtn := widget.NewButtonWithIcon("✅ Good (3)", nil, func() {
		sra.rateCard(fsrs.Good)
	})
	goodBtn.Importance = widget.SuccessImportance

	easyBtn := widget.NewButtonWithIcon("🌟 Easy (4)", nil, func() {
		sra.rateCard(fsrs.Easy)
	})
	easyBtn.Importance = widget.HighImportance

	// Rating buttons container with spacing
	sra.ratingContainer = container.NewGridWithColumns(4,
		againBtn, hardBtn, goodBtn, easyBtn)
	sra.ratingContainer.Hide()

	// Stats label with enhanced styling
	sra.statsLabel = widget.NewLabelWithStyle("No cards loaded",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Create card-like containers for better visual separation
	statsCard := container.NewPadded(sra.statsLabel)

	// Fixed-height container for question and answer to prevent jumping
	questionCard := container.NewVBox(
		container.NewPadded(sra.questionLabel),
		container.NewPadded(sra.answerLabel),
	)

	// Create a fixed container where both show answer button and rating buttons will appear
	// This prevents the UI from jumping when switching between them
	buttonContainer := container.NewStack(sra.showAnswerBtn, sra.ratingContainer)
	actionCard := container.NewPadded(buttonContainer)

	// Main content with better spacing - no load button needed
	content := container.NewVBox(
		statsCard,
		widget.NewSeparator(),
		questionCard,
		widget.NewSeparator(),
		actionCard,
	)

	// Add overall padding for a cleaner look
	sra.window.SetContent(container.NewPadded(content))

	// Setup keyboard shortcuts
	sra.setupKeyboardShortcuts()
}

func (sra *SpacedRepetitionApp) setupKeyboardShortcuts() {
	sra.window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeyS:
			// Show answer
			if sra.currentCard != nil && !sra.showingAnswer {
				sra.showAnswer()
			}
		case fyne.Key1:
			// Again rating
			if sra.currentCard != nil && sra.showingAnswer {
				sra.rateCard(fsrs.Again)
			}
		case fyne.Key2:
			// Hard rating
			if sra.currentCard != nil && sra.showingAnswer {
				sra.rateCard(fsrs.Hard)
			}
		case fyne.Key3:
			// Good rating
			if sra.currentCard != nil && sra.showingAnswer {
				sra.rateCard(fsrs.Good)
			}
		case fyne.Key4:
			// Easy rating
			if sra.currentCard != nil && sra.showingAnswer {
				sra.rateCard(fsrs.Easy)
			}
		case fyne.KeyN:
			// Add new card (Ctrl+N would be better but this is simpler)
			if sra.parser.HasFile() {
				sra.showAddCardDialog()
			}
			return // Consume the key event to prevent it from reaching other handlers
		}
	})
}

func (sra *SpacedRepetitionApp) loadCards() {
	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, sra.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		filePath := reader.URI().Path()

		sra.parser.Clear()
		if err := sra.parser.LoadFromFile(filePath); err != nil {
			dialog.ShowError(err, sra.window)
			return
		}

		// Show parse report if there were issues
		if sra.parser.HasParseErrors() {
			parseReport := sra.parser.GetParseReport()
			dialog.ShowInformation("File Parse Report", parseReport, sra.window)
		} else if sra.parser.GetCardCount() > 0 {
			// Show success message for clean parse
			result := sra.parser.GetParseResult()
			successMsg := fmt.Sprintf("✅ Successfully loaded %d cards from %d lines.",
				result.ValidCards, result.TotalLines)
			dialog.ShowInformation("Cards Loaded", successMsg, sra.window)
		}

		if err := sra.fsrsManager.LoadState(); err != nil {
			dialog.ShowError(err, sra.window)
			return
		}

		sra.updateDueCards()
		sra.updateStats()
		sra.nextCard()

	}, sra.window)

	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".txt"}))
	fileDialog.Show()
}

func (sra *SpacedRepetitionApp) updateDueCards() {
	allCards := sra.parser.GetCards()
	sra.dueCards = sra.fsrsManager.GetDueCards(allCards)
	sra.currentIndex = -1
}

func (sra *SpacedRepetitionApp) updateStats() {
	allCards := sra.parser.GetCards()
	total, due, reviewed := sra.fsrsManager.GetStats(allCards)

	if total == 0 {
		sra.statsLabel.SetText("📚 No cards loaded - Use File → Open Cards... to get started!\n💡 Supports formats: question>>answer, question::answer, question|answer")
		return
	}

	var progressEmoji string
	progressPercent := float64(reviewed) / float64(total) * 100

	switch {
	case progressPercent == 0:
		progressEmoji = "🆕"
	case progressPercent < 25:
		progressEmoji = "🌱"
	case progressPercent < 50:
		progressEmoji = "🌿"
	case progressPercent < 75:
		progressEmoji = "🌳"
	case progressPercent < 100:
		progressEmoji = "⭐"
	default:
		progressEmoji = "🏆"
	}

	var dueEmoji string
	if due == 0 {
		dueEmoji = "✅ All done!"
	} else if due <= 5 {
		dueEmoji = "📝"
	} else {
		dueEmoji = "📚"
	}

	// Add session and streak info
	streak := sra.statsManager.GetLearningStreak()
	todayStats := sra.statsManager.GetTodayStats()

	var sessionInfo string
	if sra.sessionStarted {
		sessionDuration := int(sra.statsManager.GetCurrentSessionDuration().Minutes())
		sessionInfo = fmt.Sprintf(" | ⏱️ Session: %dm", sessionDuration)
	}

	statsText := fmt.Sprintf("%s Progress: %d/%d cards (%.0f%%) | %s Due: %d%s\n🔥 Streak: %d days | 📅 Today: %d cards",
		progressEmoji, reviewed, total, progressPercent, dueEmoji, due, sessionInfo,
		streak.CurrentStreak, todayStats.CardsReviewed)

	sra.statsLabel.SetText(statsText)
}

func (sra *SpacedRepetitionApp) nextCard() {
	if len(sra.dueCards) == 0 {
		allCards := sra.parser.GetCards()
		if len(allCards) == 0 {
			sra.questionLabel.SetText("🎯 Welcome to Spaced Repetition!\n\nUse File → Open Cards... to load your first card file and start learning efficiently.\n\n⌨️ Keyboard shortcuts: S = Show Answer, 1-4 = Rate cards, N = Add card")
		} else {
			sra.questionLabel.SetText("🎉 Congratulations!\n\nAll cards reviewed for today. Come back later for more practice!")
		}
		sra.answerLabel.SetText("")
		sra.showAnswerBtn.Hide()
		sra.ratingContainer.Hide()
		sra.currentCard = nil
		return
	}

	sra.currentIndex = (sra.currentIndex + 1) % len(sra.dueCards)
	sra.currentCard = &sra.dueCards[sra.currentIndex]

	// Add card counter to question
	cardPosition := fmt.Sprintf("📊 Card %d of %d\n\n%s",
		sra.currentIndex+1, len(sra.dueCards), sra.currentCard.Question)

	sra.questionLabel.SetText(cardPosition)
	sra.answerLabel.SetText("") // Clear answer text but keep label visible
	sra.showAnswerBtn.Show()
	sra.ratingContainer.Hide()
	sra.showingAnswer = false
}

func (sra *SpacedRepetitionApp) showAnswer() {
	if sra.currentCard == nil {
		return
	}

	answerText := fmt.Sprintf("💡 Answer:\n\n%s", sra.currentCard.Answer)
	sra.answerLabel.SetText(answerText)
	sra.showAnswerBtn.Hide()
	sra.ratingContainer.Show()
	sra.showingAnswer = true
}

func (sra *SpacedRepetitionApp) rateCard(rating fsrs.Rating) {
	if sra.currentCard == nil {
		return
	}

	// Start session if not started
	if !sra.sessionStarted {
		sra.statsManager.StartSession()
		sra.sessionStarted = true
	}

	// Check if this is a new card
	cardState := sra.fsrsManager.GetCardState(*sra.currentCard)
	isNewCard := cardState.ReviewCount == 0

	// Record the review in FSRS
	if err := sra.fsrsManager.ReviewCard(*sra.currentCard, rating); err != nil {
		dialog.ShowError(err, sra.window)
		return
	}

	// Record statistics
	sra.statsManager.RecordCardReview(isNewCard)

	sra.updateDueCards()
	sra.updateStats()
	sra.nextCard()
}

func (sra *SpacedRepetitionApp) showStatistics() {
	todayStats := sra.statsManager.GetTodayStats()
	weekStats := sra.statsManager.GetWeeklyStats()
	streak := sra.statsManager.GetLearningStreak()
	totalCards, totalTime, totalSessions := sra.statsManager.GetAllTimeStats()

	// Current session info
	sessionDuration := sra.statsManager.GetCurrentSessionDuration()
	sessionInfo := ""
	if sra.sessionStarted {
		sessionStats := sra.statsManager.GetCurrentSessionStats()
		sessionInfo = fmt.Sprintf("📊 Current Session:\n- Duration: %d minutes\n- Cards reviewed: %d\n\n",
			int(sessionDuration.Minutes()), sessionStats.CardsReviewed)
	}

	// Weekly summary
	weeklyCards := 0
	weeklyTime := 0
	for _, day := range weekStats {
		weeklyCards += day.CardsReviewed
		weeklyTime += day.SessionTime
	}

	statsText := fmt.Sprintf(`%s🏆 Learning Statistics

📅 Today:
- Cards reviewed: %d
- Study time: %d minutes
- Sessions: %d

🔥 Learning Streak:
- Current streak: %d days
- Longest streak: %d days

📈 This Week:
- Cards reviewed: %d
- Study time: %d minutes

🎯 All Time:
- Total cards: %d
- Total time: %d hours
- Total sessions: %d`,
		sessionInfo,
		todayStats.CardsReviewed, todayStats.SessionTime, todayStats.SessionCount,
		streak.CurrentStreak, streak.LongestStreak,
		weeklyCards, weeklyTime,
		totalCards, totalTime/60, totalSessions)

	dialog.ShowInformation("Learning Statistics", statsText, sra.window)
}

func (sra *SpacedRepetitionApp) exportStatistics() {
	saveDialog := dialog.NewFileSave(func(writer fyne.URIWriteCloser, err error) {
		if err != nil {
			dialog.ShowError(err, sra.window)
			return
		}
		if writer == nil {
			return
		}
		defer writer.Close()

		filePath := writer.URI().Path()
		if err := sra.statsManager.ExportToCSV(filePath); err != nil {
			dialog.ShowError(fmt.Errorf("Failed to export statistics: %w", err), sra.window)
			return
		}

		dialog.ShowInformation("Export Complete",
			fmt.Sprintf("Statistics exported to:\n%s", filePath), sra.window)
	}, sra.window)

	saveDialog.SetFileName("spaced_repetition_stats.csv")
	saveDialog.Show()
}

func (sra *SpacedRepetitionApp) resetStatistics() {
	dialog.ShowConfirm("Reset Statistics",
		"Are you sure you want to reset all statistics? This cannot be undone.",
		func(confirmed bool) {
			if confirmed {
				sra.statsManager = NewStatisticsManager("./spaced_repetition_stats.json")
				sra.sessionStarted = false
				dialog.ShowInformation("Statistics Reset", "All statistics have been reset.", sra.window)
			}
		}, sra.window)
}

func (sra *SpacedRepetitionApp) showAddCardDialog() {
	if !sra.parser.HasFile() {
		dialog.ShowInformation("No File Loaded",
			"Please load a card file first using File → Open Cards...", sra.window)
		return
	}

	// Create multiline entry widget for multiple cards
	cardsEntry := widget.NewMultiLineEntry()
	cardsEntry.SetPlaceHolder("Enter cards in format:\nQuestion 1>>Answer 1\nQuestion 2>>Answer 2\n...")
	cardsEntry.Resize(fyne.NewSize(500, 200))

	// Create buttons
	addButton := widget.NewButton("Add Card", nil)
	addButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Cancel", nil)

	// Create form
	form := container.NewVBox(
		widget.NewLabelWithStyle("Add New Cards", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),
		widget.NewLabel("Cards (one per line, format: question>>answer):"),
		cardsEntry,
		widget.NewSeparator(),
		container.NewHBox(addButton, cancelButton),
	)

	// Create custom dialog window without the framework's close button
	addDialog := dialog.NewCustomWithoutButtons("Add Card", form, sra.window)

	// Function to add the cards
	addCards := func() {
		text := strings.TrimSpace(cardsEntry.Text)
		if text == "" {
			dialog.ShowError(fmt.Errorf("please enter at least one card"), sra.window)
			return
		}

		// Parse lines and add cards
		lines := strings.Split(text, "\n")
		var addedCount int
		var errors []string

		for i, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "#") {
				continue // Skip empty lines and comments
			}

			// Parse question>>answer format
			parts := strings.Split(line, ">>")
			if len(parts) != 2 {
				errors = append(errors, fmt.Sprintf("Line %d: Invalid format (use question>>answer)", i+1))
				continue
			}

			question := strings.TrimSpace(parts[0])
			answer := strings.TrimSpace(parts[1])

			if question == "" || answer == "" {
				errors = append(errors, fmt.Sprintf("Line %d: Question and answer cannot be empty", i+1))
				continue
			}

			// Add the card
			if err := sra.parser.AddCard(question, answer); err != nil {
				errors = append(errors, fmt.Sprintf("Line %d: %v", i+1, err))
				continue
			}
			addedCount++
		}

		// Update the UI
		sra.updateDueCards()
		sra.updateStats()

		// Show results
		fileName := filepath.Base(sra.parser.GetCurrentFile())
		if len(errors) > 0 {
			errorMsg := fmt.Sprintf("Added %d cards to %s\n\nErrors:\n%s",
				addedCount, fileName, strings.Join(errors, "\n"))
			dialog.ShowInformation("Cards Added with Errors", errorMsg, sra.window)
		} else {
			dialog.ShowInformation("Cards Added",
				fmt.Sprintf("Added %d cards to %s", addedCount, fileName), sra.window)
		}

		// Close dialog
		addDialog.Hide()
	}

	// Set button actions
	addButton.OnTapped = addCards
	cancelButton.OnTapped = func() {
		addDialog.Hide()
	}

	// Add custom key handling for Ctrl+Enter to submit
	cardsEntry.OnSubmitted = func(text string) {
		// This won't be called for multiline entries, but we keep it for consistency
	}

	// Store original key handler
	originalSetup := sra.setupKeyboardShortcuts

	// Custom key event handling for tab navigation and Ctrl+Enter submission
	sra.window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeyTab:
			// Handle tab navigation
			focused := sra.window.Canvas().Focused()
			if focused == cardsEntry {
				sra.window.Canvas().Focus(addButton)
				return
			} else if focused == addButton {
				sra.window.Canvas().Focus(cancelButton)
				return
			} else if focused == cancelButton {
				sra.window.Canvas().Focus(cardsEntry)
				return
			}
		case fyne.KeyReturn, fyne.KeyEnter:
			// Handle Enter on buttons
			focused := sra.window.Canvas().Focused()
			if focused == addButton {
				addCards()
				return
			} else if focused == cancelButton {
				addDialog.Hide()
				return
			}
		case fyne.KeyEscape:
			// Escape to cancel
			addDialog.Hide()
			return
		default:
			// For other keys, check if it's study-related and handle appropriately
			if key.Name == fyne.KeyS || key.Name == fyne.Key1 || key.Name == fyne.Key2 ||
			   key.Name == fyne.Key3 || key.Name == fyne.Key4 {
				// Ignore study shortcuts while in dialog
				return
			}
		}
	})

	// Restore original key handler when dialog closes
	addDialog.SetOnClosed(func() {
		originalSetup()
	})

	addDialog.Resize(fyne.NewSize(500, 450))
	addDialog.Show()

	// Focus on cards field and clear any stray characters
	sra.window.Canvas().Focus(cardsEntry)

	// Use a short delay to clear any stray keystrokes that opened the dialog
	go func() {
		time.Sleep(50 * time.Millisecond)
		fyne.Do(func() {
			cardsEntry.SetText("")
		})
	}()
}

func (sra *SpacedRepetitionApp) Run() {
	// End session when window closes
	sra.window.SetOnClosed(func() {
		if sra.sessionStarted {
			sra.statsManager.EndSession()
		}
	})
	sra.window.ShowAndRun()
}

func main() {
	app := NewSpacedRepetitionApp()
	app.setupUI()

	// Load statistics
	if err := app.statsManager.LoadStats(); err != nil {
		log.Printf("Failed to load statistics: %v", err)
	}

	if err := app.parser.LoadFromFile("sample_cards.txt"); err != nil {
		log.Printf("Failed to load sample cards: %v", err)
	} else {
		if err := app.fsrsManager.LoadState(); err != nil {
			log.Printf("Failed to load state: %v", err)
		}
		app.updateDueCards()
		app.updateStats()
		app.nextCard()
	}

	app.window.ShowAndRun()
}