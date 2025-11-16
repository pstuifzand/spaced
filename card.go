package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"
	"unicode/utf8"
)

type Card struct {
	ID            int64     // Database ID (0 for file-based cards)
	Question      string
	Answer        string
	FilePath      string
	LineNum       int
	SourceContext string    // Book, article, project name
	PromptType    string    // factual, conceptual, application, comparison
	Tags          string    // Comma-separated tags
	CreatedAt     time.Time // When the card was created
}

type ParseError struct {
	LineNum int
	Line    string
	Reason  string
}

type ParseResult struct {
	Cards       []Card
	Errors      []ParseError
	TotalLines  int
	ValidCards  int
	SkippedLines int
}

type CardParser struct {
	cards       []Card
	parseResult *ParseResult
	currentFile string
	cardRepo    CardRepository
}

func NewCardParserWithDatabase(cardRepo CardRepository) *CardParser {
	return &CardParser{
		cards:    make([]Card, 0),
		cardRepo: cardRepo,
	}
}

func (cp *CardParser) LoadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	// Store current file path
	cp.currentFile = filePath

	// Initialize parse result
	cp.parseResult = &ParseResult{
		Cards:  make([]Card, 0),
		Errors: make([]ParseError, 0),
	}

	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		cp.parseResult.TotalLines++

		line := scanner.Text()

		// Check for valid UTF-8
		if !utf8.ValidString(line) {
			cp.parseResult.Errors = append(cp.parseResult.Errors, ParseError{
				LineNum: lineNum,
				Line:    line,
				Reason:  "Invalid UTF-8 encoding",
			})
			cp.parseResult.SkippedLines++
			continue
		}

		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}

		// Try multiple separators
		separators := []string{">>", "::", "|"}
		var parts []string

		for _, sep := range separators {
			if strings.Contains(line, sep) {
				parts = strings.Split(line, sep)
				break
			}
		}

		if len(parts) != 2 {
			cp.parseResult.Errors = append(cp.parseResult.Errors, ParseError{
				LineNum: lineNum,
				Line:    line,
				Reason:  fmt.Sprintf("No valid separator found. Expected one of: %s", strings.Join(separators, ", ")),
			})
			cp.parseResult.SkippedLines++
			continue
		}

		question := strings.TrimSpace(parts[0])
		answer := strings.TrimSpace(parts[1])

		// Validate question and answer
		if question == "" {
			cp.parseResult.Errors = append(cp.parseResult.Errors, ParseError{
				LineNum: lineNum,
				Line:    line,
				Reason:  "Empty question part",
			})
			cp.parseResult.SkippedLines++
			continue
		}

		if answer == "" {
			cp.parseResult.Errors = append(cp.parseResult.Errors, ParseError{
				LineNum: lineNum,
				Line:    line,
				Reason:  "Empty answer part",
			})
			cp.parseResult.SkippedLines++
			continue
		}

		// Check for extremely long content (might indicate parsing error)
		if len(question) > 1000 || len(answer) > 1000 {
			cp.parseResult.Errors = append(cp.parseResult.Errors, ParseError{
				LineNum: lineNum,
				Line:    line,
				Reason:  "Question or answer exceeds 1000 characters - possible parsing error",
			})
			cp.parseResult.SkippedLines++
			continue
		}

		card := Card{
			Question: question,
			Answer:   answer,
			FilePath: filePath,
			LineNum:  lineNum,
		}

		// Store in memory for immediate access
		cp.cards = append(cp.cards, card)
		cp.parseResult.Cards = append(cp.parseResult.Cards, card)
		cp.parseResult.ValidCards++

		// Store in database
		if cp.cardRepo != nil {
			// Check if card already exists to avoid duplicates
			exists, err := cp.cardRepo.CardExists(question, answer)
			if err != nil {
				cp.parseResult.Errors = append(cp.parseResult.Errors, ParseError{
					LineNum: lineNum,
					Line:    line,
					Reason:  fmt.Sprintf("Failed to check card existence: %v", err),
				})
			} else if !exists {
				// Only import if card doesn't exist
				_, err := cp.cardRepo.ImportFromText(question, answer, filePath, lineNum)
				if err != nil {
					// Log error but continue processing other cards
					cp.parseResult.Errors = append(cp.parseResult.Errors, ParseError{
						LineNum: lineNum,
						Line:    line,
						Reason:  fmt.Sprintf("Database import failed: %v", err),
					})
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	return nil
}

func (cp *CardParser) GetCards() []Card {
	// Load cards from database
	if cp.cardRepo != nil {
		dbCards, err := cp.cardRepo.GetAll()
		if err != nil {
			// Fall back to in-memory cards if database fails
			return cp.cards
		}

		// Convert DB cards to Card structs
		var cards []Card
		for _, dbCard := range dbCards {
			sourceContext := ""
			if dbCard.SourceContext.Valid {
				sourceContext = dbCard.SourceContext.String
			}
			card := Card{
				ID:            dbCard.ID,
				Question:      dbCard.Question,
				Answer:        dbCard.Answer,
				FilePath:      dbCard.SourceFile,
				LineNum:       dbCard.SourceLine,
				SourceContext: sourceContext,
				PromptType:    dbCard.PromptType,
				Tags:          dbCard.Tags,
				CreatedAt:     dbCard.CreatedAt,
			}
			cards = append(cards, card)
		}
		return cards
	}

	return cp.cards
}

func (cp *CardParser) GetCardCount() int {
	// Get count from database
	if cp.cardRepo != nil {
		cards := cp.GetCards()
		return len(cards)
	}
	return len(cp.cards)
}

func (cp *CardParser) GetParseResult() *ParseResult {
	return cp.parseResult
}

func (cp *CardParser) GetParseReport() string {
	if cp.parseResult == nil {
		return "No file has been parsed yet."
	}

	report := fmt.Sprintf("Parse Summary:\n")
	report += fmt.Sprintf("- Total lines processed: %d\n", cp.parseResult.TotalLines)
	report += fmt.Sprintf("- Valid cards created: %d\n", cp.parseResult.ValidCards)
	report += fmt.Sprintf("- Lines skipped: %d\n", cp.parseResult.SkippedLines)

	if len(cp.parseResult.Errors) > 0 {
		report += fmt.Sprintf("\nParsing Issues (%d):\n", len(cp.parseResult.Errors))
		for i, err := range cp.parseResult.Errors {
			if i >= 10 { // Limit to first 10 errors
				report += fmt.Sprintf("... and %d more errors\n", len(cp.parseResult.Errors)-10)
				break
			}
			line := err.Line
			if len(line) > 50 {
				line = line[:47] + "..."
			}
			report += fmt.Sprintf("  Line %d: %s - %s\n", err.LineNum, line, err.Reason)
		}
	}

	return report
}

func (cp *CardParser) HasParseErrors() bool {
	return cp.parseResult != nil && len(cp.parseResult.Errors) > 0
}

func (cp *CardParser) AddCard(question, answer string) error {
	return cp.AddCardWithMetadata(question, answer, "", "factual", "")
}

func (cp *CardParser) AddCardWithMetadata(question, answer, source, promptType, tags string) error {
	if question == "" || answer == "" {
		return fmt.Errorf("question and answer cannot be empty")
	}

	// Check if card already exists
	if cp.cardRepo != nil {
		exists, err := cp.cardRepo.CardExists(question, answer)
		if err != nil {
			return fmt.Errorf("failed to check if card exists: %w", err)
		}
		if exists {
			return fmt.Errorf("card with this question and answer already exists")
		}

		// Add to database with metadata
		sourceContext := sql.NullString{String: source, Valid: source != ""}
		dbCard := &DBCard{
			Question:      question,
			Answer:        answer,
			SourceFile:    cp.currentFile,
			SourceLine:    len(cp.cards) + 1,
			SourceContext: sourceContext,
			PromptType:    promptType,
			Tags:          tags,
		}
		err = cp.cardRepo.Create(dbCard)
		if err != nil {
			return fmt.Errorf("failed to add card to database: %w", err)
		}
	}

	// Create new card for memory cache
	newCard := Card{
		Question:      question,
		Answer:        answer,
		FilePath:      cp.currentFile,
		LineNum:       len(cp.cards) + 1,
		SourceContext: source,
		PromptType:    promptType,
		Tags:          tags,
		CreatedAt:     time.Now(),
	}

	// Add to memory
	cp.cards = append(cp.cards, newCard)

	return nil
}

func (cp *CardParser) UpdateCard(cardID int64, question, answer string) error {
	if question == "" || answer == "" {
		return fmt.Errorf("question and answer cannot be empty")
	}

	if cp.cardRepo == nil {
		return fmt.Errorf("no database repository available")
	}

	// Get the existing card
	existingCard, err := cp.cardRepo.GetByID(cardID)
	if err != nil {
		return fmt.Errorf("failed to get card: %w", err)
	}

	// Update the card
	existingCard.Question = question
	existingCard.Answer = answer
	existingCard.UpdatedAt = time.Now()

	err = cp.cardRepo.Update(existingCard)
	if err != nil {
		return fmt.Errorf("failed to update card: %w", err)
	}

	return nil
}

func (cp *CardParser) DeleteCard(cardID int64) error {
	if cp.cardRepo == nil {
		return fmt.Errorf("no database repository available")
	}

	err := cp.cardRepo.Delete(cardID)
	if err != nil {
		return fmt.Errorf("failed to delete card: %w", err)
	}

	return nil
}

func (cp *CardParser) GetCurrentFile() string {
	return cp.currentFile
}

func (cp *CardParser) HasFile() bool {
	return cp.currentFile != ""
}

func (cp *CardParser) Clear() {
	cp.cards = cp.cards[:0]
	cp.parseResult = nil
	cp.currentFile = ""
}