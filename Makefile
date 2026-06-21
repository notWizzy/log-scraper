BINARY    := logscraper
MODULE    := github.com/notWizzy/log-scraper
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT    := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD     := $(shell date -u '+%Y-%m-%dT%H:%M:%SZ')
LDFLAGS   := -s -w -X main.version=$(VERSION) -X main.commit=$(COMMIT) -X main.buildDate=$(BUILD)

.PHONY: build run test bench vet clean release

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/logscraper

run: build
	./bin/$(BINARY) $(ARGS)

test:
	go test -race -count=1 ./...

bench:
	go test -bench=. -benchmem ./internal/matcher/ ./internal/scanner/

vet:
	go vet ./...

clean:
	rm -rf bin/

release:
	GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-amd64   ./cmd/logscraper
	GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-linux-arm64   ./cmd/logscraper
	GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-darwin-amd64  ./cmd/logscraper
	GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY)-darwin-arm64  ./cmd/logscraper
