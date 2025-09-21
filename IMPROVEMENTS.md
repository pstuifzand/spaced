# Spaced Repetition App Improvements TODO

## Phase 1: File Input Robustness (HIGH PRIORITY)

### Better Error Handling in Card Parser
- [ ] Add line-by-line validation with detailed error reporting
- [ ] Show which lines were skipped and why (malformed, empty, etc.)
- [ ] Continue parsing even when encountering bad lines
- [ ] Add support for multiple separators (`>>`, `::`, `|`)
- [ ] Add line number reporting in error messages

### Enhanced File Validation
- [ ] UTF-8 encoding detection and support
- [ ] File size warnings for very large files
- [ ] Backup original files before processing
- [ ] Better error messages with suggestions
- [ ] Handle files with mixed line endings

## Phase 2: Enhanced Statistics (HIGH PRIORITY)

### Persistent Statistics System
- [ ] Daily/weekly/monthly review tracking
- [ ] Session duration measurement
- [ ] Learning streak counters
- [ ] Statistics persistence across app restarts
- [ ] Review history with timestamps

### Card Performance Analysis
- [ ] Track difficulty patterns per card
- [ ] Review interval effectiveness analysis
- [ ] Identify consistently problematic cards
- [ ] Export statistics to CSV for personal analysis
- [ ] Show learning curve over time

## Phase 3: Code Stability (HIGH PRIORITY)

### Better Error Recovery
- [ ] Graceful handling of corrupted state files
- [ ] Auto-rebuild mechanism for damaged data
- [ ] Safe shutdown procedures that always save state
- [ ] Memory usage optimization for large card sets
- [ ] Prevent data loss on unexpected crashes

## Phase 4: Quality of Life (MEDIUM PRIORITY)

### Session Management
- [ ] Remember window position and size
- [ ] Auto-load last used card file
- [ ] Quick session restart functionality
- [ ] Simple in-app card editing
- [ ] Session pause/resume functionality

### User Convenience Features
- [ ] Search functionality within cards
- [ ] Duplicate card detection with warnings
- [ ] Manual card scheduling override
- [ ] Daily study goal setting with progress tracking
- [ ] Card bookmarking for problem cards

## Current Status
Starting with Phase 1 - improving file input robustness for better daily reliability.

## Notes
- Focus on personal use case - reliability over features
- Maintain simple, clean interface
- Don't over-engineer - practical improvements only
- Test with real card files during development