# AGENTS.md - Developer Guide

## Project Overview

### Attributes
Project name: raw_view
Main file: raw_view.go

### Intent and Functionality
This is a Go-based terminal file viewer application that visualizes binary files in multiple modes:
- **Hexdump**: Traditional hex view with 16 bytes per row
    - Includes detection of embedded file via magic
- **Linear**: Wrapped linear byte grid
- **Hilbert Curve**: Space-filling curve visualization

This application uses the Bubble Tea TUI framework to handle rending and updates and supports multiple color-coded byte visualization.

Interaction is handled via the keyboard and includes the following functionality:
- Scrolling by displayed row via arrow up and down keys
- Scrolling by displayed page via page up and page down keys
- Jumping to a specific byte offset via the j key
- Searching for a text string via the s key
- Switching visualization modes via the tab key
- Switching byte-value color coding scheme via the / key
- Exiting the application via escape key or q key

## Build Commands
```bash
# Build the application
go build -o raw_view .

# Run the application
go run . <filename>

# Format code
go fmt ./...

# Run linter (vet)
go vet ./...

# Run all tests
go test -v ./...

# Run a single test
go test -v -run TestFunctionName ./...

# Run tests with coverage
go test -v -coverprofile=coverage.out ./...

# View coverage report
go tool cover -html=coverage.out
```

## Best Practices

### Planning
- Always use the provided skill `create-task` to create tasks and subtasks for requested changes and update them as they are completed

### Code Organization and Structure
- Use clear package structure
- Follow go naming conventions
- Write idiomatic go
- Prioritize short, single-purpose, testable functions
- Prefer functional paradigms
- Enforce pre-conditions
- Implement strict error handling
- Write concise, function-level comments
- Always validate formatting using `go format`
- Utilize yaml configuration files to store configuration
- Store bulk data in data files instead of inline where applicable

### Testing Strategy
- Include integration tests
- Mock external dependencies when possible
- Maintain code coverage
- Always vet code `go vet`

### Documentation
- Always update the README.md file (or create if it doesn't exist) to accurately describe the project and provided basic instructions for users.

### Version control
- When all tasks are complete, stage all modified files in git and suggest a commit message
