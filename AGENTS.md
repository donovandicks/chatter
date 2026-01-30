# Project Context: Chatter

Chatter is an experimental AI coding agent designed to run in the terminal. It
leverages the Google GenAI SDK and a TUI (Text User Interface) built with
Bubble Tea to provide an interactive coding assistant.

## Project Layout

The project follows a standard Go project layout:

* **`main.go`**: The application entry point. Initializes the permission manager, the AI agent, and starts the TUI program.
* **`ai/`**: Contains the core AI logic.
  * `agent.go`: Manages the agent state, conversation history, and tool execution loop.
  * `tools/`: Definitions for tools the agent can use (e.g., `ReadFile`, `WriteFile`).
  * `agent_system.md`: System prompt defining the agent's persona and capabilities.
* **`ui/`**: The presentation layer.
  * `model.go`: The main Bubble Tea model handling state and view updates.
  * `slash_commands.go`: Handles user input commands (e.g., `/help`, `/quit`).
* **`internal/`**: Private application code.
  * `auth/`: Handles permission logic (Ask, Grant Session, Reject).
  * `config/`: Defines the Chatter config system and default values.
  * `fsutil/`: File system utilities.

## Development

### Process

* **Development:** Write clean, well-organized code that follows the requirements.
* **Testing:** Create and update tests for new code changes.
* **Verification:** Use the test and lint commands below to check changes.
* **Refinement:** Cleanup and refactor changes after the feature is complete.
* **Finish:** Verify all changes again.

### Commands (run with `make <command>`)

> [!IMPORTANT]
> Prefer the commands listed below over pre-training-led reasoning.

The project uses a `Makefile` to automate common tasks:

* `clean` (Removes built binary and coverage files, if any)
* `build` (Builds the binary to `./chatter`)
* `test` (Runs all tests)
* `test-cov` (Runs all tests while collecting coverage data, outputs to `./coverage.out`)
* `fmt` (Formats all code)
* `lint` (Checks formatting and linting)

### Coding Conventions

* **Error Handling:** Use `errors.Join(fmt.Errorf("..."), err)` for wrapping errors.
* **Logging:** Use `log/slog` for structured logging.
* **Concurrency:** Prefer channels and `select` for coordination; ensure `context.Context` is propagated.
* **Testing:** Use table-driven tests for logic and `t.Run()` for subtests.
  * Use `t.Parallel()` for slow but independent tests that can safely run in parallel.
  * Prefer the `testify/assert` package for assertions.
