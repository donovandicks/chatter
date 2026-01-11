.PHONY: clean build test test-cov fmt lint

clean:
	rm -f "./chatter" && rm -f "./coverage.out"

build:
	go build -o chatter main.go

test:
	go test ./...

test-cov:
	go test -coverprofile=coverage.out ./...

fmt:
	gofumpt -w .

lint:
	gofumpt -d . && golangci-lint run ./...
