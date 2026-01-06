# Chatter

Chatter is an experimental AI coding agent project.

## Tooling

The project uses the Go programming language, using the latest 1.25 version.
It uses modern Golang conventions and the `go` CLI command for build and test,
though there are some convenient recipes in `Makefile` for common commands.

The project uses the popular "github.com/charmbracelet/bubbletea" TUI library
for the terminal interface.

## AI Usage

The project exclusively (for now) uses Google's Gemini AI. The AI API is
accessed using the `google.golang.org/genai` SDK.

## Guidelines

- Use only modern, idiomatic Golang.
- Write tests for functions when applicable.
  - Use table-driven tests when appropriate.

## Rules

Run the following commands and fix any issues before committing changes:

- `make fmt`
- `make lint`
- `make test`
- `make build`

Commit changes after writing code with `git commit -m "<YOUR MESSAGE HERE>"`. Commit
messages should start with a short subject line followed by a large description
on a newline.
