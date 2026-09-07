BINARY := wake
PKG := github.com/DiegoHeer/wakelab
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X $(PKG)/internal/cli.version=$(VERSION)

COVER_FLOOR := 80

.PHONY: build test cover cover-check integration lint fmt install clean

build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/wake

test:
	go test -race ./...

cover:
	go test -race -coverprofile=coverage.txt -covermode=atomic ./internal/...
	go tool cover -func=coverage.txt | tail -1

cover-check: cover
	@go tool cover -func=coverage.txt | awk -v floor=$(COVER_FLOOR) \
		'/^total:/ { gsub("%","",$$3); if ($$3+0 < floor) { printf "FAIL: coverage %.1f%% is below the %d%% floor\n", $$3, floor; exit 1 } \
		else printf "coverage %.1f%% (floor %d%%)\n", $$3, floor }'

integration: build
	sh tests/integration.sh ./bin/$(BINARY)

lint:
	golangci-lint run

fmt:
	gofmt -w .

install: build
	mkdir -p $(HOME)/.local/bin
	cp bin/$(BINARY) $(HOME)/.local/bin/$(BINARY)

clean:
	rm -rf bin dist coverage.txt
