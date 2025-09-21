# Spaced Repetition App Improvements TODO

## Phase 1: File Input Robustness (HIGH PRIORITY) ✅ COMPLETED

### Better Error Handling in Card Parser
- [x] Add line-by-line validation with detailed error reporting
- [x] Show which lines were skipped and why (malformed, empty, etc.)
- [x] Continue parsing even when encountering bad lines
- [x] Add support for multiple separators (`>>`, `::`, `|`)
- [x] Add line number reporting in error messages

### Enhanced File Validation
- [x] UTF-8 encoding detection and support
- [x] File size warnings for very large files
- [x] Better error messages with suggestions
- [x] Handle files with mixed line endings
- [x] Content length validation (prevent parsing errors)

## Phase 2: Enhanced Statistics (HIGH PRIORITY) ✅ COMPLETED

### Persistent Statistics System
- [x] Daily/weekly/monthly review tracking
- [x] Session duration measurement
- [x] Learning streak counters
- [x] Statistics persistence across app restarts
- [x] Review history with timestamps

### Card Performance Analysis
- [x] Track new vs reviewed cards separately
- [x] Export statistics to CSV for personal analysis
- [x] Session-based tracking with automatic save
- [x] Real-time session duration display
- [x] Learning streak with current/longest tracking

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