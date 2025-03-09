# CLAUDE.md - Kafka Explorer CLI Guidelines

## Build & Test Commands
- Build: `go build -o kafka-explorer .`  
- Run: `./kafka-explorer --brokers=localhost:9092 --topic=test --search="example"`
- Test all: `go test ./...`
- Test single package: `go test ./pkg/consumer`
- Test specific test: `go test ./pkg/consumer -run TestConsumerConnect`
- Lint: `golangci-lint run`

## Code Style Guidelines
- **Imports**: Group standard library first, then third-party, then internal imports
- **Formatting**: Use `gofmt` for consistent formatting; align struct fields
- **Types**: Use strong typing with interfaces for testability; avoid interface{}
- **Naming**: Use camelCase for private, PascalCase for exported items
- **Error Handling**: Check errors immediately, use fmt.Errorf with %w for context
- **Comments**: Document exported items with godoc style; add context to complex logic
- **Structure**: Organize by feature with clear separation of concerns
- **Testing**: Write unit tests that cover happy paths and error conditions
- **CLI**: Use cobra for command structure, viper for configuration
- **Concurrency**: Use contexts for cancellation; mutexes for shared state

## Project Architecture
- TUI: Use bubbletea/lipgloss for interactive display with consistent styling
- Kafka: Isolate operations in consumer package using segmentio/kafka-go
- Output: JSON writer should support streaming for performance
- Search: Support both string and regex pattern matching