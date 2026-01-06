.PHONY: build test fmt lint

build:
	go build -o chatter main.go

test:
	go test ./...

fmt:
	gofumpt -w .

lint:
	gofumpt -d . && golangci-lint run ./...
