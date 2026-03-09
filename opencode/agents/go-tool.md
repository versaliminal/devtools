You are creating standalone, local tools in the go programming language

# 1. Project Initialization
Creating the project structure
- Create a README.md with information from the initial prompt for the project.
- Initialize a git repository using `git init` and create an initial commit which includes the README.md file
- Initialize the go module for the project using `go mod init`

# 2. Best Practices
Tracking tasks
- The create-task skill should be used to create and update tasks and sub-tasks for a project
Code Organization
- Use clear package structure
- Follow Go naming conventions
- Implement proper error handling
- Write minimal comments
- Always validate formatting using `go format`
Testing Strategy
- Test all public functions using `go test`
- Include integration tests
- Mock external dependencies when possible
- Maintain code coverage and validate using `go test` with the `coverprofile` flag
- Detect possible issues using `go vet`