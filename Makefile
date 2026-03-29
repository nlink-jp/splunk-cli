VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := splunk-cli
CMD     := ./cmd/splunk-cli

.PHONY: build test vet lint check build-all clean \
        splunk-up splunk-down integration-test

build: _dist
	go build $(LDFLAGS) -o dist/$(BINARY) $(CMD)

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

splunk-up:
	@eval "$$(scripts/splunk-up.sh)" && \
		printf '\nSplunk is up. To set env vars in your shell:\n' && \
		printf '  eval "$$(scripts/splunk-up.sh)"\n\n'

splunk-down:
	scripts/splunk-down.sh

## Run integration tests against a live Splunk container.
## Starts Splunk automatically if not already running; leaves it running afterwards.
## Use 'make splunk-down' to tear it down when done.
integration-test:
	@if ! podman container exists splunk-test 2>/dev/null || \
	    [ "$$(podman inspect --format '{{.State.Status}}' splunk-test 2>/dev/null)" != "running" ]; then \
		echo "[integration-test] Starting Splunk container..."; \
		eval "$$(scripts/splunk-up.sh)"; \
	else \
		echo "[integration-test] Container already running."; \
	fi
	@HOST=$$(podman port splunk-test 8089/tcp | cut -d: -f2) && \
		TOKEN=$$(curl -sk \
			-d "username=admin&password=Admin1234!&output_mode=json" \
			"https://localhost:$${HOST}/services/auth/login" \
			| python3 -c "import sys,json; print(json.load(sys.stdin)['sessionKey'])") && \
		SPLUNK_HOST="https://localhost:$${HOST}" \
		SPLUNK_TOKEN="$${TOKEN}" \
		go test -v -tags integration -timeout 5m ./internal/client/...

clean:
	rm -rf dist/
