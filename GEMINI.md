# Chatter

Chatter is an experimental AI coding agent project.

## Tooling & Environment

- **Go Version:** 1.25+
- **Linter:** `golangci-lint` (comprehensive static analysis)
- **Formatter:** `gofumpt` (strict "gofmt")
- **TUI Framework:** `charmbracelet/bubbletea` (Elm architecture)
- **AI SDK:** `google.golang.org/genai`

## Best Practices

### Code Quality & Style

- **Error Handling:** Use `fmt.Errorf("...: %w", err)` for wrapping errors to preserve context. Handle all errors explicitly.
- **Context:** Propagate `context.Context` as the first argument in long-running or I/O-bound functions.
- **Logging:** Use `log/slog` for structured, leveled logging.
- **Concurrency:** Prefer channels and `select` for orchestration; use `sync` primitives for state protection. Avoid uncaught goroutines.
- **Configuration:** Use functional options for complex constructors.

### Architecture

- **Package Layout:** Keep core logic in `internal/`. Expose only necessary API surfaces.
- **Interfaces:** Define small, consumer-centric interfaces (Interface Segregation Principle).
- **TUI (Bubble Tea):** Keep `Update` functions pure where possible. Use `tea.Cmd` for all side effects (I/O, API calls).

### Testing

- **Table-Driven Tests:** Strongly preferred for logic with multiple edge cases.
- **Subtests:** Use `t.Run()` for clear test hierarchy.
- **Test Helpers:** Mark helpers with `t.Helper()` for accurate failure reporting.

## Workflow

1. Develop: Implement changes and features as requested.
2. **Verify:** Run `make fmt lint test build` before committing.
3. **Commit:** Use the conventional format:

    ```text
    <short summary>

    <detailed description if necessary>
    ```

