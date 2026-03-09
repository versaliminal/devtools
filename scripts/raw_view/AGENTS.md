# AGENTS.md - Developer Guide

This document provides guidelines for agentic coding agents working in this repository.

## Project Overview
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
Tracking tasks
- Always use the create-task skill to create and update tasks and sub-tasks for a project
Code Organization
- Use clear package structure
- Follow Go naming conventions
- Implement proper error handling
- Write minimal comments
- Always validate formatting using `go format`
Testing Strategy
- Include integration tests
- Mock external dependencies when possible
- Maintain code coverage
- Always vet code `go vet`