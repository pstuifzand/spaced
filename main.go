package main

import (
	"fmt"
	"log"

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

	currentCard    *Card
	currentIndex   int
	dueCards      []Card

	questionLabel  *widget.Label
	answerLabel    *widget.Label
	showAnswerBtn  *widget.Button
	ratingContainer *fyne.Container
	statsLabel     *widget.Label

	showingAnswer  bool
}

func NewSpacedRepetitionApp() *SpacedRepetitionApp {
	myApp := app.New()
	myApp.SetIcon(nil)
	myApp.Settings().SetTheme(&SpacedRepetitionTheme{})

	window := myApp.NewWindow("Spaced Repetition - Learn Efficiently")
	window.Resize(fyne.NewSize(900, 700))
	window.CenterOnScreen()

	sra := &SpacedRepetitionApp{
		app:        myApp,
		window:     window,
		parser:     NewCardParser(),
		fsrsManager: NewFSRSManager("./spaced_repetition_state.json"),
		currentIndex: -1,
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

	// Create menu items
	fileMenu := fyne.NewMenu("File",
		openCards,
	)

	// Create Help menu
	helpMenu := fyne.NewMenu("Help",
		fyne.NewMenuItem("About", func() {
			dialog.ShowInformation("About",
				"Spaced Repetition v1.0\n\nAn efficient learning tool using the FSRS algorithm.\n\nLoad cards in 'question>>answer' format and study efficiently!",
				sra.window)
		}),
	)

	// Create main menu
	mainMenu := fyne.NewMainMenu(fileMenu, helpMenu)
	sra.window.SetMainMenu(mainMenu)
}

func (sra *SpacedRepetitionApp) setupUI() {
	// Question label - large and prominent
	sra.questionLabel = widget.NewLabelWithStyle("Welcome to Spaced Repetition!",
		fyne.TextAlignCenter, fyne.TextStyle{Bold: true})
	sra.questionLabel.Wrapping = fyne.TextWrapWord

	// Answer label - clear but secondary
	sra.answerLabel = widget.NewLabelWithStyle("",
		fyne.TextAlignCenter, fyne.TextStyle{})
	sra.answerLabel.Wrapping = fyne.TextWrapWord
	sra.answerLabel.Hide()

	// Show answer button - prominent
	sra.showAnswerBtn = widget.NewButton("👁️ Show Answer", sra.showAnswer)
	sra.showAnswerBtn.Importance = widget.HighImportance
	sra.showAnswerBtn.Hide()

	// Color-coded rating buttons with icons
	againBtn := widget.NewButtonWithIcon("❌ Again", nil, func() {
		sra.rateCard(fsrs.Again)
	})
	againBtn.Importance = widget.DangerImportance

	hardBtn := widget.NewButtonWithIcon("⚠️ Hard", nil, func() {
		sra.rateCard(fsrs.Hard)
	})
	hardBtn.Importance = widget.MediumImportance

	goodBtn := widget.NewButtonWithIcon("✅ Good", nil, func() {
		sra.rateCard(fsrs.Good)
	})
	goodBtn.Importance = widget.SuccessImportance

	easyBtn := widget.NewButtonWithIcon("🌟 Easy", nil, func() {
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

	questionCard := container.NewVBox(
		container.NewPadded(sra.questionLabel),
		container.NewPadded(sra.answerLabel),
	)

	actionCard := container.NewVBox(
		container.NewPadded(sra.showAnswerBtn),
		container.NewPadded(sra.ratingContainer),
	)

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
		sra.statsLabel.SetText("📚 No cards loaded - Use File → Open Cards... to get started!")
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

	statsText := fmt.Sprintf("%s Progress: %d/%d cards (%.0f%%) | %s Due: %d",
		progressEmoji, reviewed, total, progressPercent, dueEmoji, due)

	sra.statsLabel.SetText(statsText)
}

func (sra *SpacedRepetitionApp) nextCard() {
	if len(sra.dueCards) == 0 {
		allCards := sra.parser.GetCards()
		if len(allCards) == 0 {
			sra.questionLabel.SetText("🎯 Welcome to Spaced Repetition!\n\nUse File → Open Cards... to load your first card file and start learning efficiently.")
		} else {
			sra.questionLabel.SetText("🎉 Congratulations!\n\nAll cards reviewed for today. Come back later for more practice!")
		}
		sra.answerLabel.Hide()
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
	sra.answerLabel.Hide()
	sra.showAnswerBtn.Show()
	sra.ratingContainer.Hide()
	sra.showingAnswer = false
}

func (sra *SpacedRepetitionApp) showAnswer() {
	if sra.currentCard == nil {
		return
	}

	answerText := fmt.Sprintf("💡 Answer:\n\n%s\n\n🤔 How well did you remember this?", sra.currentCard.Answer)
	sra.answerLabel.SetText(answerText)
	sra.answerLabel.Show()
	sra.showAnswerBtn.Hide()
	sra.ratingContainer.Show()
	sra.showingAnswer = true
}

func (sra *SpacedRepetitionApp) rateCard(rating fsrs.Rating) {
	if sra.currentCard == nil {
		return
	}

	if err := sra.fsrsManager.ReviewCard(*sra.currentCard, rating); err != nil {
		dialog.ShowError(err, sra.window)
		return
	}

	sra.updateDueCards()
	sra.updateStats()
	sra.nextCard()
}

func (sra *SpacedRepetitionApp) Run() {
	sra.window.ShowAndRun()
}

func main() {
	app := NewSpacedRepetitionApp()
	app.setupUI()

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