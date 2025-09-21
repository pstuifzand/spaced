package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"unicode/utf8"
)

type Card struct {
	Question string
	Answer   string
	FilePath string
	LineNum  int
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
}

func NewCardParser() *CardParser {
	return &CardParser{
		cards: make([]Card, 0),
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

		cp.cards = append(cp.cards, card)
		cp.parseResult.Cards = append(cp.parseResult.Cards, card)
		cp.parseResult.ValidCards++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading file %s: %w", filePath, err)
	}

	return nil
}

func (cp *CardParser) GetCards() []Card {
	return cp.cards
}

func (cp *CardParser) GetCardCount() int {
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
	if question == "" || answer == "" {
		return fmt.Errorf("question and answer cannot be empty")
	}

	if cp.currentFile == "" {
		return fmt.Errorf("no file loaded - please load a card file first")
	}

	// Create new card
	newCard := Card{
		Question: question,
		Answer:   answer,
		FilePath: cp.currentFile,
		LineNum:  len(cp.cards) + 1, // Approximate line number
	}

	// Add to memory
	cp.cards = append(cp.cards, newCard)

	// Append to file
	return cp.appendCardToFile(question, answer)
}

func (cp *CardParser) appendCardToFile(question, answer string) error {
	file, err := os.OpenFile(cp.currentFile, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("failed to open file for writing: %w", err)
	}
	defer file.Close()

	// Write the new card with >> separator
	cardLine := fmt.Sprintf("%s>>%s\n", question, answer)
	_, err = file.WriteString(cardLine)
	if err != nil {
		return fmt.Errorf("failed to write to file: %w", err)
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