VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := splunk-cli
CMD     := ./cmd/splunk-cli

.PHONY: build test vet lint check build-all clean

build:
	go build $(LDFLAGS) -o $(BINARY) $(CMD)

test:
	go test ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

check: vet lint test build

build-all: _dist
	GOOS=linux   GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-amd64   $(CMD)
	GOOS=linux   GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-linux-arm64   $(CMD)
	GOOS=darwin  GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-amd64  $(CMD)
	GOOS=darwin  GOARCH=arm64  go build $(LDFLAGS) -o dist/$(BINARY)-darwin-arm64  $(CMD)
	GOOS=windows GOARCH=amd64  go build $(LDFLAGS) -o dist/$(BINARY)-windows-amd64.exe $(CMD)
	@if command -v lipo >/dev/null 2>&1; then \
		lipo -create -output dist/$(BINARY)-darwin-universal \
			dist/$(BINARY)-darwin-amd64 dist/$(BINARY)-darwin-arm64; \
		echo "Universal macOS binary: dist/$(BINARY)-darwin-universal"; \
	fi

_dist:
	mkdir -p dist

clean:
	rm -f $(BINARY)
	rm -rf dist/
