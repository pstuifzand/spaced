package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/open-spaced-repetition/go-fsrs/v3"
)

type SpacedRepetitionApp struct {
	app          fyne.App
	window       fyne.Window
	parser       *CardParser
	fsrsManager  *FSRSManager
	statsManager *StatisticsManager
	database     *Database

	currentCard          *Card
	currentIndex         int
	dueCards             []Card
	sessionCardsReviewed int
	initialDueCount      int

	questionLabel   *widget.Label
	answerLabel     *widget.Label
	showAnswerBtn   *widget.Button
	ratingContainer *fyne.Container
	statsLabel      *widget.Label

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

	// Initialize database (required for operation)
	database, err := NewDatabase("./spaced_repetition.db")
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize database: %v", err))
	}

	// Create repositories
	cardRepo := NewSQLiteCardRepository(database)
	reviewRepo := NewSQLiteReviewStateRepository(database)
	sessionRepo := NewSQLiteSessionRepository(database)
	dailyStatsRepo := NewSQLiteDailyStatsRepository(database)

	sra := &SpacedRepetitionApp{
		app:                  myApp,
		window:               window,
		parser:               NewCardParserWithDatabase(cardRepo),
		fsrsManager:          NewFSRSManagerWithDatabase(reviewRepo),
		statsManager:         NewStatisticsManagerWithDatabase(sessionRepo, dailyStatsRepo),
		database:             database,
		currentIndex:         -1,
		sessionCardsReviewed: 0,
		initialDueCount:      0,
		sessionStarted:       false,
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

	manageCards := fyne.NewMenuItem("Manage Cards...", func() {
		sra.showCardManagementDialog()
	})

	exportStats := fyne.NewMenuItem("Export Statistics...", func() {
		sra.exportStatistics()
	})

	quitApp := fyne.NewMenuItem("Quit", func() {
		sra.quit()
	})

	// Create menu items
	fileMenu := fyne.NewMenu("File",
		openCards,
		fyne.NewMenuItemSeparator(),
		addCard,
		manageCards,
		fyne.NewMenuItemSeparator(),
		exportStats,
		fyne.NewMenuItemSeparator(),
		quitApp,
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
		sra.resetSession()
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

func (sra *SpacedRepetitionApp) resetSession() {
	sra.sessionCardsReviewed = 0
	sra.currentIndex = -1
	sra.initialDueCount = len(sra.dueCards)
	fmt.Printf("DEBUG: resetSession called - initialDueCount set to %d\n", sra.initialDueCount)
}

func (sra *SpacedRepetitionApp) updateDueCardsKeepSession() {
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

	// Build context information
	var contextInfo string
	if sra.currentCard.SourceContext != "" || !sra.currentCard.CreatedAt.IsZero() {
		contextParts := []string{}
		if sra.currentCard.SourceContext != "" {
			contextParts = append(contextParts, fmt.Sprintf("📚 %s", sra.currentCard.SourceContext))
		}
		if !sra.currentCard.CreatedAt.IsZero() {
			dateStr := sra.currentCard.CreatedAt.Format("2006-01-02")
			contextParts = append(contextParts, fmt.Sprintf("Added %s", dateStr))
		}
		contextInfo = fmt.Sprintf("[%s]\n\n", strings.Join(contextParts, " • "))
	}

	// Display remaining cards and question with context
	remaining := len(sra.dueCards)
	cardPosition := fmt.Sprintf("📊 %d cards remaining\n\n%s%s",
		remaining, contextInfo, sra.currentCard.Question)

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

	// Add prompt type indicator if available
	promptTypeIndicator := ""
	switch sra.currentCard.PromptType {
	case "conceptual":
		promptTypeIndicator = "🧠 Conceptual"
	case "application":
		promptTypeIndicator = "⚙️ Application"
	case "comparison":
		promptTypeIndicator = "⚖️ Comparison"
	case "factual":
		promptTypeIndicator = "📝 Factual"
	}

	answerHeader := "💡 Answer"
	if promptTypeIndicator != "" {
		answerHeader = fmt.Sprintf("💡 Answer (%s)", promptTypeIndicator)
	}

	answerText := fmt.Sprintf("%s:\n\n%s", answerHeader, sra.currentCard.Answer)
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
	if !sra.statsManager.HasActiveSession() {
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

	// Increment session counter
	sra.sessionCardsReviewed++

	sra.updateDueCardsKeepSession()
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
				// Create fresh repositories for database mode
				sessionRepo := NewSQLiteSessionRepository(sra.database)
				dailyStatsRepo := NewSQLiteDailyStatsRepository(sra.database)
				sra.statsManager = NewStatisticsManagerWithDatabase(sessionRepo, dailyStatsRepo)
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

	// Create input fields
	questionEntry := widget.NewMultiLineEntry()
	questionEntry.SetPlaceHolder("Enter your question...")
	questionEntry.Wrapping = fyne.TextWrapWord
	questionEntry.SetMinRowsVisible(3)

	answerEntry := widget.NewMultiLineEntry()
	answerEntry.SetPlaceHolder("Enter the answer...")
	answerEntry.Wrapping = fyne.TextWrapWord
	answerEntry.SetMinRowsVisible(3)

	sourceEntry := widget.NewEntry()
	sourceEntry.SetPlaceHolder("Book, article, or project (optional)")

	tagsEntry := widget.NewEntry()
	tagsEntry.SetPlaceHolder("e.g., #golang #algorithms (optional)")

	// Prompt type radio buttons
	var promptType string = "conceptual"
	promptTypeGroup := widget.NewRadioGroup([]string{
		"Factual Recall",
		"Conceptual",
		"Application",
		"Comparison",
	}, func(value string) {
		promptType = value
	})
	promptTypeGroup.SetSelected("Conceptual")
	promptTypeGroup.Horizontal = false

	// Tips based on prompt type
	tipLabel := widget.NewLabel("💡 Tip: Conceptual prompts build deeper understanding than simple definitions. Ask 'why' and 'how' questions.")
	tipLabel.Wrapping = fyne.TextWrapWord

	// Update tip when prompt type changes
	promptTypeGroup.OnChanged = func(value string) {
		switch value {
		case "Factual Recall":
			tipLabel.SetText("💡 Tip: Use for basic facts and definitions. Keep questions atomic and focused on one piece of information.")
		case "Conceptual":
			tipLabel.SetText("💡 Tip: Ask 'why' and 'how' questions to build deeper understanding. Focus on relationships and mechanisms.")
		case "Application":
			tipLabel.SetText("💡 Tip: Ask 'when would you use X?' or 'give an example of X in context Y'. Promotes transfer of knowledge.")
		case "Comparison":
			tipLabel.SetText("💡 Tip: Ask about differences and similarities. Helps build relational understanding between concepts.")
		}
	}

	// Create buttons
	addButton := widget.NewButton("Add Card", nil)
	addButton.Importance = widget.HighImportance

	addAnotherButton := widget.NewButton("Add & Create Another", nil)
	cancelButton := widget.NewButton("Cancel", nil)

	// Create form layout
	form := container.NewVBox(
		widget.NewLabelWithStyle("Add New Card", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),

		widget.NewLabel("Question:"),
		questionEntry,

		widget.NewLabel("Answer:"),
		answerEntry,

		widget.NewSeparator(),
		widget.NewLabel("Prompt Type:"),
		promptTypeGroup,
		container.NewPadded(tipLabel),

		widget.NewSeparator(),
		widget.NewLabel("Source (optional):"),
		sourceEntry,

		widget.NewLabel("Tags (optional):"),
		tagsEntry,

		widget.NewSeparator(),
		container.NewHBox(addButton, addAnotherButton, cancelButton),
	)

	// Create custom dialog window without the framework's close button
	addDialog := dialog.NewCustomWithoutButtons("Add Card", form, sra.window)

	// Function to add the card
	addCard := func(closeDialog bool) {
		question := strings.TrimSpace(questionEntry.Text)
		answer := strings.TrimSpace(answerEntry.Text)

		if question == "" {
			dialog.ShowError(fmt.Errorf("question cannot be empty"), sra.window)
			return
		}
		if answer == "" {
			dialog.ShowError(fmt.Errorf("answer cannot be empty"), sra.window)
			return
		}

		source := strings.TrimSpace(sourceEntry.Text)
		tags := strings.TrimSpace(tagsEntry.Text)

		// Map prompt type display name to internal value
		promptTypeValue := "factual"
		switch promptType {
		case "Factual Recall":
			promptTypeValue = "factual"
		case "Conceptual":
			promptTypeValue = "conceptual"
		case "Application":
			promptTypeValue = "application"
		case "Comparison":
			promptTypeValue = "comparison"
		}

		// Add the card with new fields
		if err := sra.parser.AddCardWithMetadata(question, answer, source, promptTypeValue, tags); err != nil {
			dialog.ShowError(fmt.Errorf("failed to add card: %w", err), sra.window)
			return
		}

		// Update the UI
		sra.updateDueCards()
		sra.updateStats()

		if closeDialog {
			dialog.ShowInformation("Card Added",
				"Card has been successfully added.", sra.window)
			addDialog.Hide()
		} else {
			// Clear fields for next card
			questionEntry.SetText("")
			answerEntry.SetText("")
			sourceEntry.SetText("")
			tagsEntry.SetText("")
			// Keep prompt type and focus on question field
			sra.window.Canvas().Focus(questionEntry)
		}
	}

	// Set button actions
	addButton.OnTapped = func() {
		addCard(true)
	}
	addAnotherButton.OnTapped = func() {
		addCard(false)
	}
	cancelButton.OnTapped = func() {
		addDialog.Hide()
	}

	// Store original key handler
	originalSetup := sra.setupKeyboardShortcuts

	// Custom key event handling for tab navigation and shortcuts
	sra.window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
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

	addDialog.Resize(fyne.NewSize(600, 700))
	addDialog.Show()

	// Focus on question field
	sra.window.Canvas().Focus(questionEntry)
}

func (sra *SpacedRepetitionApp) showCardManagementDialog() {
	// Get all cards from database
	var allCards []Card
	var filteredCards []Card
	var searchEntry *widget.Entry
	var cardContainer *fyne.Container
	var scrollableList *container.Scroll

	refreshCards := func() {
		oldCount := len(allCards)
		allCards = sra.parser.GetCards()
		filteredCards = allCards
		fmt.Printf("DEBUG: refreshCards - old count: %d, new count: %d\n", oldCount, len(allCards))
	}

	// Function to recreate the card list entirely
	updateList := func() {
		if searchEntry == nil {
			filteredCards = allCards
		} else {
			searchText := strings.ToLower(strings.TrimSpace(searchEntry.Text))
			if searchText == "" {
				filteredCards = allCards
			} else {
				filteredCards = nil
				for _, card := range allCards {
					if strings.Contains(strings.ToLower(card.Question), searchText) ||
						strings.Contains(strings.ToLower(card.Answer), searchText) {
						filteredCards = append(filteredCards, card)
					}
				}
			}
		}

		if cardContainer != nil {
			fmt.Printf("DEBUG: updateList - filteredCards count: %d\n", len(filteredCards))
			// Clear and recreate the container contents
			cardContainer.RemoveAll()

			// Add each card as a separate widget
			for _, card := range filteredCards {
				cardWidget := sra.createCardWidget(card, func() {
					// Refresh callback for deletion - reload cards and refresh display
					refreshCards()
					// Force container update by clearing and rebuilding
					cardContainer.RemoveAll()
					for _, newCard := range filteredCards {
						newWidget := sra.createCardWidget(newCard, nil) // Pass nil to avoid infinite recursion
						cardContainer.Add(newWidget)
					}
					cardContainer.Refresh()
				})
				cardContainer.Add(cardWidget)
			}

			cardContainer.Refresh()
		}
	}

	refreshCards()
	if len(allCards) == 0 {
		dialog.ShowInformation("No Cards", "No cards are currently loaded.", sra.window)
		return
	}

	// Create a scrollable container for cards
	cardContainer = container.NewVBox()
	scrollableList = container.NewScroll(cardContainer)
	scrollableList.SetMinSize(fyne.NewSize(700, 400))

	// Create search entry with better styling
	searchEntry = widget.NewEntry()
	searchEntry.SetPlaceHolder("🔍 Search cards by question or answer...")

	searchEntry.OnChanged = func(string) {
		updateList()
	}

	// Create header with stats
	cardCount := len(allCards)
	headerText := fmt.Sprintf("Card Management - %d cards loaded", cardCount)
	headerLabel := widget.NewLabelWithStyle(headerText, fyne.TextAlignCenter, fyne.TextStyle{Bold: true})

	// Create dialog content with better proportions
	content := container.NewBorder(
		// Top: Header and search
		container.NewVBox(
			headerLabel,
			widget.NewSeparator(),
			searchEntry,
			widget.NewSeparator(),
		),
		// Bottom: nothing
		nil,
		// Left: nothing
		nil,
		// Right: nothing
		nil,
		// Center: scrollable list
		scrollableList,
	)

	// Initialize the list with all cards
	updateList()

	// Create larger dialog
	manageDialog := dialog.NewCustom("Manage Cards", "Close", content, sra.window)
	manageDialog.Resize(fyne.NewSize(800, 600))
	manageDialog.Show()
}

func (sra *SpacedRepetitionApp) createCardWidget(card Card, refreshCallback func()) fyne.CanvasObject {
	// Create larger, more prominent labels
	questionLabel := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	questionLabel.Wrapping = fyne.TextWrapWord

	answerLabel := widget.NewLabel("")
	answerLabel.TextStyle.Italic = true
	answerLabel.Wrapping = fyne.TextWrapWord

	// Show more text with better formatting - increase character limits
	question := card.Question
	if len(question) > 200 {
		question = question[:197] + "..."
	}
	answer := card.Answer
	if len(answer) > 200 {
		answer = answer[:197] + "..."
	}

	questionLabel.SetText(fmt.Sprintf("📝 %s", question))
	answerLabel.SetText(fmt.Sprintf("💡 %s", answer))

	// Create more prominent buttons
	editBtn := widget.NewButtonWithIcon("✏️ Edit", nil, func() {
		sra.showEditCardDialog(card.ID, card.Question, card.Answer)
	})
	editBtn.Importance = widget.MediumImportance

	deleteBtn := widget.NewButtonWithIcon("🗑️ Delete", nil, func() {
		if refreshCallback != nil {
			sra.confirmDeleteCardFromManagement(card.ID, card.Question, refreshCallback)
		} else {
			// Fallback delete without refresh (should not be used much)
			sra.confirmDeleteCard(card.ID, card.Question)
		}
	})
	deleteBtn.Importance = widget.DangerImportance

	buttonContainer := container.NewHBox(
		editBtn,
		widget.NewSeparator(),
		deleteBtn,
	)

	// Create a padded container for better spacing
	cardWidget := container.NewVBox(
		container.NewPadded(questionLabel),
		container.NewPadded(answerLabel),
		container.NewPadded(buttonContainer),
		widget.NewSeparator(),
	)

	return cardWidget
}

func (sra *SpacedRepetitionApp) showEditCardDialog(cardID int64, currentQuestion, currentAnswer string) {
	// Create multiline entry widgets for question and answer
	questionEntry := widget.NewMultiLineEntry()
	questionEntry.SetText(currentQuestion)
	questionEntry.Wrapping = fyne.TextWrapWord

	answerEntry := widget.NewMultiLineEntry()
	answerEntry.SetText(currentAnswer)
	answerEntry.Wrapping = fyne.TextWrapWord

	// Create character count labels
	questionCount := widget.NewLabel(fmt.Sprintf("Characters: %d", len(currentQuestion)))
	answerCount := widget.NewLabel(fmt.Sprintf("Characters: %d", len(currentAnswer)))

	// Update character counts on text change
	questionEntry.OnChanged = func(text string) {
		questionCount.SetText(fmt.Sprintf("Characters: %d", len(text)))
	}
	answerEntry.OnChanged = func(text string) {
		answerCount.SetText(fmt.Sprintf("Characters: %d", len(text)))
	}

	// Create buttons
	saveButton := widget.NewButton("Save Changes", nil)
	saveButton.Importance = widget.HighImportance

	cancelButton := widget.NewButton("Cancel", nil)

	// Create form content
	form := container.NewVBox(
		widget.NewLabelWithStyle("Edit Card", fyne.TextAlignCenter, fyne.TextStyle{Bold: true}),
		widget.NewSeparator(),

		widget.NewLabel("Question:"),
		questionEntry,
		questionCount,

		widget.NewSeparator(),

		widget.NewLabel("Answer:"),
		answerEntry,
		answerCount,

		widget.NewSeparator(),
		container.NewHBox(saveButton, cancelButton),
	)

	// Create dialog
	editDialog := dialog.NewCustomWithoutButtons("Edit Card", form, sra.window)

	// Save function
	saveCard := func() {
		question := strings.TrimSpace(questionEntry.Text)
		answer := strings.TrimSpace(answerEntry.Text)

		if question == "" {
			dialog.ShowError(fmt.Errorf("question cannot be empty"), sra.window)
			return
		}
		if answer == "" {
			dialog.ShowError(fmt.Errorf("answer cannot be empty"), sra.window)
			return
		}

		// Update the card
		if err := sra.parser.UpdateCard(cardID, question, answer); err != nil {
			dialog.ShowError(fmt.Errorf("failed to update card: %w", err), sra.window)
			return
		}

		// Refresh the UI
		sra.updateDueCards()
		sra.updateStats()

		dialog.ShowInformation("Card Updated", "Card has been successfully updated.", sra.window)
		editDialog.Hide()
	}

	// Set button actions
	saveButton.OnTapped = saveCard
	cancelButton.OnTapped = func() {
		editDialog.Hide()
	}

	// Set up keyboard shortcuts
	originalSetup := sra.setupKeyboardShortcuts
	sra.window.Canvas().SetOnTypedKey(func(key *fyne.KeyEvent) {
		switch key.Name {
		case fyne.KeyEscape:
			editDialog.Hide()
			return
		case fyne.KeyTab:
			// Handle tab navigation
			focused := sra.window.Canvas().Focused()
			if focused == questionEntry {
				sra.window.Canvas().Focus(answerEntry)
				return
			} else if focused == answerEntry {
				sra.window.Canvas().Focus(saveButton)
				return
			} else if focused == saveButton {
				sra.window.Canvas().Focus(cancelButton)
				return
			} else if focused == cancelButton {
				sra.window.Canvas().Focus(questionEntry)
				return
			}
		case fyne.KeyReturn, fyne.KeyEnter:
			focused := sra.window.Canvas().Focused()
			if focused == saveButton {
				saveCard()
				return
			} else if focused == cancelButton {
				editDialog.Hide()
				return
			}
		}
	})

	// Restore original key handler when dialog closes
	editDialog.SetOnClosed(func() {
		originalSetup()
	})

	editDialog.Resize(fyne.NewSize(500, 600))
	editDialog.Show()

	// Focus on question field
	sra.window.Canvas().Focus(questionEntry)
}

func (sra *SpacedRepetitionApp) confirmDeleteCard(cardID int64, question string) {
	// Truncate question for display in confirmation
	displayQuestion := question
	if len(displayQuestion) > 100 {
		displayQuestion = displayQuestion[:97] + "..."
	}

	message := fmt.Sprintf("Are you sure you want to delete this card?\n\nQuestion: %s\n\nThis action cannot be undone and will also remove any associated review data.", displayQuestion)

	dialog.ShowConfirm("Delete Card", message, func(confirmed bool) {
		if confirmed {
			sra.deleteCard(cardID)
		}
	}, sra.window)
}

func (sra *SpacedRepetitionApp) deleteCard(cardID int64) {
	// Delete the FSRS review state first (if it exists)
	if err := sra.fsrsManager.DeleteCardState(cardID); err != nil {
		// Log but don't fail - the review state might not exist
		fmt.Printf("Warning: Failed to delete review state for card %d: %v\n", cardID, err)
	}

	// Delete the card
	if err := sra.parser.DeleteCard(cardID); err != nil {
		dialog.ShowError(fmt.Errorf("failed to delete card: %w", err), sra.window)
		return
	}

	// Update the UI
	sra.updateDueCards()
	sra.updateStats()
	sra.nextCard() // Move to next card if current card was deleted
}

func (sra *SpacedRepetitionApp) confirmDeleteCardFromManagement(cardID int64, question string, refreshCallback func()) {
	// Truncate question for display in confirmation
	displayQuestion := question
	if len(displayQuestion) > 100 {
		displayQuestion = displayQuestion[:97] + "..."
	}

	message := fmt.Sprintf("Are you sure you want to delete this card?\n\nQuestion: %s\n\nThis action cannot be undone and will also remove any associated review data.", displayQuestion)

	dialog.ShowConfirm("Delete Card", message, func(confirmed bool) {
		if confirmed {
			sra.deleteCardFromManagement(cardID, refreshCallback)
		}
	}, sra.window)
}

func (sra *SpacedRepetitionApp) deleteCardFromManagement(cardID int64, refreshCallback func()) {
	// Delete the FSRS review state first (if it exists)
	if err := sra.fsrsManager.DeleteCardState(cardID); err != nil {
		// Log but don't fail - the review state might not exist
		fmt.Printf("Warning: Failed to delete review state for card %d: %v\n", cardID, err)
	}

	// Delete the card
	if err := sra.parser.DeleteCard(cardID); err != nil {
		dialog.ShowError(fmt.Errorf("failed to delete card: %w", err), sra.window)
		return
	}

	// Update the main UI
	sra.updateDueCards()
	sra.updateStats()
	sra.nextCard() // Move to next card if current card was deleted

	// Call the refresh callback to update the management dialog
	if refreshCallback != nil {
		refreshCallback()
	}
}

func (sra *SpacedRepetitionApp) quit() {
	fmt.Printf("Quit method called - HasActiveSession: %v\n", sra.statsManager.HasActiveSession())
	if sra.statsManager.HasActiveSession() {
		fmt.Println("Ending session from quit method")
		sra.statsManager.EndSession()
	}
	// Close database if initialized
	if sra.database != nil {
		sra.database.Close()
	}
	sra.app.Quit()
}

func (sra *SpacedRepetitionApp) Run() {
	// End session when window closes
	sra.window.SetOnClosed(func() {
		fmt.Printf("Window close handler called - HasActiveSession: %v\n", sra.statsManager.HasActiveSession())
		if sra.statsManager.HasActiveSession() {
			fmt.Println("Ending session from window close handler")
			sra.statsManager.EndSession()
		}
		// Close database if initialized
		if sra.database != nil {
			sra.database.Close()
		}
	})
	sra.window.ShowAndRun()
}

func main() {
	app := NewSpacedRepetitionApp()
	app.setupUI()

	// Set up signal handling for graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		log.Println("Received termination signal, ending session...")
		if app.statsManager.HasActiveSession() {
			app.statsManager.EndSession()
		}
		if app.database != nil {
			app.database.Close()
		}
		os.Exit(0)
	}()

	// Perform one-time migration if using database
	if app.database != nil {
		// Ensure JSON files exist for legacy support
		if err := EnsureJSONFilesExist(); err != nil {
			log.Printf("Failed to ensure JSON files exist: %v", err)
		}

		// Backup existing JSON files before migration
		if err := BackupJSONFiles(); err != nil {
			log.Printf("Failed to backup JSON files: %v", err)
		}

		// Migrate existing JSON data to database
		if err := MigrateJSONToDatabase(app.database); err != nil {
			log.Printf("Failed to migrate data to database: %v", err)
		}

		// Clean up orphaned sessions from previous app instances
		if err := app.statsManager.CleanupOrphanedSessions(); err != nil {
			log.Printf("Failed to cleanup orphaned sessions: %v", err)
		}
	}

	// Load sample cards if available
	if err := app.parser.LoadFromFile("sample_cards.txt"); err != nil {
		log.Printf("Failed to load sample cards: %v", err)
	} else {
		app.updateDueCards()
		app.updateStats()
		app.nextCard()
	}

	app.window.ShowAndRun()
}

