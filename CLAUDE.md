# CLAUDE.md - AI Assistant Guide for Spaced Repository

> **Last Updated**: 2025-11-14
> **Repository**: pstuifzand/spaced
> **Purpose**: [To be defined - appears to be a new project]

This document provides comprehensive guidance for AI assistants (like Claude) working on this codebase. It explains the project structure, development workflows, coding conventions, and best practices.

---

## Table of Contents

1. [Project Overview](#project-overview)
2. [Repository Structure](#repository-structure)
3. [Development Workflows](#development-workflows)
4. [Coding Conventions](#coding-conventions)
5. [AI Assistant Guidelines](#ai-assistant-guidelines)
6. [Common Tasks](#common-tasks)
7. [Testing & Quality](#testing--quality)
8. [Deployment](#deployment)
9. [Troubleshooting](#troubleshooting)

---

## Project Overview

### Purpose
[This section should be updated once the project purpose is defined]

The "spaced" repository appears to be a new project. Based on the name, it may be related to:
- Spaced repetition learning systems
- Scheduling or spacing algorithms
- Time-based data management
- [Update with actual purpose]

### Key Technologies
[To be updated as the tech stack is established]

- Programming Language: [TBD]
- Framework: [TBD]
- Database: [TBD]
- Build Tools: [TBD]

### Project Goals
[To be defined]

---

## Repository Structure

### Current State
This is a new repository. The structure will be documented as it evolves.

### Expected Directory Layout
```
spaced/
├── src/              # Source code
├── tests/            # Test files
├── docs/             # Documentation
├── config/           # Configuration files
├── scripts/          # Build and utility scripts
├── .github/          # GitHub workflows and templates
├── CLAUDE.md         # This file
└── README.md         # Project README
```

### Key Directories & Files
[To be updated as the codebase grows]

---

## Development Workflows

### Git Branching Strategy

**Branch Naming Convention**:
- Feature branches: `feature/descriptive-name`
- Bug fixes: `fix/issue-description`
- Claude AI branches: `claude/claude-md-{session-id}` (auto-generated)

**Workflow**:
1. Create feature branch from main branch
2. Make changes and commit frequently
3. Write clear, descriptive commit messages
4. Push to remote repository
5. Create pull request for review

### Commit Message Guidelines

Follow conventional commits format:
```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types**:
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks
- `perf`: Performance improvements

**Examples**:
```
feat(auth): add user authentication system

Implement JWT-based authentication with refresh tokens.
Includes login, logout, and token validation.

Closes #123
```

```
fix(api): handle null response in user endpoint

Add null checks to prevent crashes when API returns empty data.
```

### Pull Request Process

1. Ensure all tests pass
2. Update documentation if needed
3. Add descriptive PR title and description
4. Link related issues
5. Request review if applicable
6. Address review feedback
7. Merge after approval

---

## Coding Conventions

### General Principles

1. **Readability**: Code should be self-documenting with clear variable/function names
2. **DRY**: Don't Repeat Yourself - extract common logic into reusable functions
3. **KISS**: Keep It Simple, Stupid - prefer simple solutions over complex ones
4. **SOLID**: Follow SOLID principles for object-oriented code
5. **Security**: Always validate inputs, sanitize outputs, handle errors gracefully

### Code Style
[To be defined based on chosen language/framework]

**General Guidelines**:
- Use consistent indentation (spaces or tabs - to be defined)
- Maximum line length: 100-120 characters
- Use meaningful variable and function names
- Add comments for complex logic only
- Keep functions small and focused (single responsibility)

### Error Handling

- Always handle errors gracefully
- Provide meaningful error messages
- Log errors appropriately
- Don't expose sensitive information in error messages
- Use try-catch blocks where appropriate

### Security Best Practices

**Never commit**:
- API keys, tokens, or credentials
- `.env` files with secrets
- Private keys or certificates
- Database passwords

**Always**:
- Use environment variables for sensitive data
- Validate and sanitize all user inputs
- Prevent SQL injection, XSS, CSRF attacks
- Keep dependencies updated
- Follow principle of least privilege

---

## AI Assistant Guidelines

### When Working on This Codebase

#### 1. Always Start By Understanding Context

- Read relevant documentation first
- Explore the codebase structure using Task/Explore agent
- Check existing patterns and conventions
- Look for similar implementations to maintain consistency

#### 2. Use TodoWrite for Task Planning

For any non-trivial task:
```
1. Create todo list with clear, actionable items
2. Mark items as in_progress when starting
3. Mark items as completed immediately after finishing
4. Keep only ONE item in_progress at a time
```

Example:
- Research existing authentication code
- Design new feature architecture
- Implement core functionality
- Write tests
- Update documentation
- Commit and push changes

#### 3. Code Quality Standards

**Before Writing Code**:
- Understand the requirements completely
- Check for existing similar functionality
- Plan the implementation approach
- Consider edge cases and error scenarios

**While Writing Code**:
- Follow existing code patterns and conventions
- Write clean, readable code with clear names
- Add comments only where logic is complex
- Handle errors appropriately
- Think about security implications

**After Writing Code**:
- Review for potential bugs or issues
- Check for security vulnerabilities
- Verify tests pass
- Update relevant documentation
- Create clear commit messages

#### 4. File Operations Best Practices

- **ALWAYS prefer editing over creating new files**
- Read files before editing them
- Use Edit tool for modifications, not Bash commands
- Never create unnecessary files (especially .md files unless requested)
- Maintain existing file structure

#### 5. Testing Approach

- Run existing tests before making changes
- Write tests for new functionality
- Ensure all tests pass before committing
- Test edge cases and error conditions
- Don't mark tasks complete if tests fail

#### 6. Git Operations

**Committing**:
- Only commit when explicitly asked or when work is complete
- Write clear, descriptive commit messages
- Use conventional commit format
- Stage only relevant files
- Never commit secrets or credentials

**Pushing**:
- Always push to Claude-specific branches (`claude/*`)
- Use `git push -u origin <branch-name>`
- Retry up to 4 times with exponential backoff on network errors
- Never force push to main/master

#### 7. Communication Style

- Be concise and professional
- Don't use emojis unless requested
- Output text directly, don't use echo or comments to communicate
- Provide file paths with line numbers (e.g., `src/auth.ts:42`)
- Be objective and accurate over validating user beliefs

#### 8. Tool Usage

**Prefer specialized tools**:
- `Read` instead of `cat/head/tail`
- `Edit` instead of `sed/awk`
- `Write` instead of `echo >` or `cat << EOF`
- `Grep` instead of `grep/rg` commands
- `Glob` instead of `find/ls` for file patterns

**Use Task agent for**:
- Exploring codebase structure
- Multi-step research tasks
- Finding files/patterns when uncertain
- Complex search operations

**Run tools in parallel when**:
- Tasks are independent
- No dependencies between operations
- Multiple files to read/search

#### 9. Security Mindset

Always check for:
- Command injection vulnerabilities
- SQL injection risks
- XSS (Cross-Site Scripting) vulnerabilities
- CSRF (Cross-Site Request Forgery) issues
- Authentication/authorization bypasses
- Sensitive data exposure
- Insecure dependencies

If you write insecure code, **immediately fix it**.

#### 10. Problem-Solving Approach

1. **Understand**: Analyze the problem thoroughly
2. **Research**: Explore existing code and patterns
3. **Plan**: Break down into manageable tasks (use TodoWrite)
4. **Implement**: Write clean, tested code
5. **Verify**: Test functionality and edge cases
6. **Document**: Update relevant documentation
7. **Review**: Check for issues, security problems, bugs
8. **Commit**: Create clear commits with good messages

---

## Common Tasks

### Setting Up Development Environment

[To be updated once project setup is defined]

```bash
# Example commands (to be updated)
git clone <repository-url>
cd spaced
# Install dependencies
# Setup configuration
# Run initial setup scripts
```

### Running the Application

[To be updated]

```bash
# Development mode
# Production mode
# With specific configuration
```

### Running Tests

[To be updated]

```bash
# Run all tests
# Run specific test suite
# Run with coverage
# Run in watch mode
```

### Building for Production

[To be updated]

```bash
# Build command
# Optimization steps
# Output verification
```

### Database Operations

[To be updated if applicable]

```bash
# Run migrations
# Seed database
# Backup/restore
# Reset database
```

---

## Testing & Quality

### Testing Strategy
[To be defined]

**Test Types**:
- Unit tests: Test individual functions/components
- Integration tests: Test component interactions
- End-to-end tests: Test complete user workflows
- Performance tests: Test system performance under load

### Code Quality Tools
[To be updated as tools are chosen]

- Linter: [TBD]
- Formatter: [TBD]
- Type checker: [TBD]
- Security scanner: [TBD]

### Coverage Requirements
[To be defined]

- Minimum coverage: [TBD]%
- Critical paths: 100% coverage
- New code: Must include tests

---

## Deployment

### Environments
[To be defined]

- **Development**: Local development environment
- **Staging**: Pre-production testing environment
- **Production**: Live production environment

### Deployment Process
[To be updated]

1. Ensure all tests pass
2. Build production bundle
3. Run deployment scripts
4. Verify deployment
5. Monitor for issues

### Environment Variables
[To be documented]

Required environment variables:
- [TBD]

---

## Troubleshooting

### Common Issues

[To be populated as issues are discovered and resolved]

#### Issue: [Problem Description]
**Symptoms**: [What you observe]
**Cause**: [Root cause]
**Solution**: [How to fix]

### Debug Mode

[To be updated]

```bash
# Enable debug logging
# Access debug tools
# View detailed errors
```

### Getting Help

1. Check this CLAUDE.md file
2. Review project README.md
3. Search existing issues
4. Check commit history for context
5. Ask project maintainers

---

## Maintenance Notes

### Updating This Document

This document should be updated when:
- New features or patterns are established
- Development workflows change
- New conventions are adopted
- Common issues are discovered
- Project structure evolves

**AI Assistants**: When you make significant changes to the codebase, consider whether CLAUDE.md needs updates to reflect new patterns or conventions.

### Version History

- **2025-11-14**: Initial creation - empty repository baseline
- [Future updates to be logged here]

---

## Quick Reference

### Essential Commands
```bash
# Git operations
git status                                    # Check status
git add .                                     # Stage all changes
git commit -m "type(scope): message"         # Commit with message
git push -u origin <branch-name>             # Push to branch

# [Add project-specific commands as they're established]
```

### Important Files
- `CLAUDE.md` - This file (AI assistant guide)
- `README.md` - Project overview and documentation
- [To be updated]

### Key Contacts
- Repository: pstuifzand/spaced
- [Add maintainer information when available]

---

## Notes for Future Development

This CLAUDE.md file was created for an empty repository. As the project grows:

1. **Update the Project Overview** with actual purpose and goals
2. **Document the tech stack** once technologies are chosen
3. **Fill in coding conventions** based on chosen languages/frameworks
4. **Add specific commands** for development, testing, and deployment
5. **Document architecture patterns** as they emerge
6. **Add troubleshooting entries** based on real issues encountered
7. **Include examples** of good code patterns to follow
8. **Update directory structure** as it evolves

**Remember**: This document is a living guide. Keep it updated to maximize its value for AI assistants and human developers alike.
