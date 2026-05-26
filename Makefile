VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"
BINARY  := splunk-cli
CMD     := .

# macOS Developer ID signing / notarization (see nlink-jp/.github
# CONVENTIONS.md §Code Signing). Defaults match any Developer ID
# Application cert in the keychain and the org-standard notary
# profile. Builds without these fall back to ad-hoc / un-notarized
# with a one-line warning — see scripts/codesign-darwin.sh.
CODESIGN_IDENTITY ?= Developer ID Application
NOTARY_PROFILE    ?= nlink-jp-notary

.PHONY: build test vet lint check build-all package clean \
        splunk-up splunk-down integration-test

build: _dist
	go build $(LDFLAGS) -o dist/$(BINARY) $(CMD)
	@scripts/codesign-darwin.sh dist/$(BINARY) "$(CODESIGN_IDENTITY)"

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
	@scripts/codesign-darwin.sh dist/$(BINARY)-darwin-amd64 "$(CODESIGN_IDENTITY)"
	@scripts/codesign-darwin.sh dist/$(BINARY)-darwin-arm64 "$(CODESIGN_IDENTITY)"
	@if [ -f dist/$(BINARY)-darwin-universal ]; then \
		scripts/codesign-darwin.sh dist/$(BINARY)-darwin-universal "$(CODESIGN_IDENTITY)"; \
	fi

## package: Build all platforms, zip with version suffix + README, notarize darwin → dist/
package: build-all
	@cd dist && for f in $(BINARY)-linux-amd64 $(BINARY)-linux-arm64 $(BINARY)-darwin-amd64 $(BINARY)-darwin-arm64 $(BINARY)-darwin-universal $(BINARY)-windows-amd64.exe; do \
		[ -f "$$f" ] || continue; \
		suffix=$${f#$(BINARY)-}; \
		suffix=$${suffix%%.exe}; \
		case "$$f" in *.exe) ext=.exe ;; *) ext= ;; esac; \
		cp ../README.md .; \
		stage="$$(dirname "$$f")/_pkg"; rm -rf "$$stage"; mkdir -p "$$stage"; \
		cp "$$f" "$$stage/$(BINARY)$$ext"; \
		zip -j "$(BINARY)-$(VERSION)-$${suffix}.zip" "$$stage/$(BINARY)$$ext" README.md; \
		rm -rf "$$stage"; \
		rm -f README.md; \
	done
	@scripts/notarize-darwin.sh dist/$(BINARY)-$(VERSION)-darwin-amd64.zip "$(NOTARY_PROFILE)"
	@scripts/notarize-darwin.sh dist/$(BINARY)-$(VERSION)-darwin-arm64.zip "$(NOTARY_PROFILE)"
	@if [ -f dist/$(BINARY)-$(VERSION)-darwin-universal.zip ]; then \
		scripts/notarize-darwin.sh dist/$(BINARY)-$(VERSION)-darwin-universal.zip "$(NOTARY_PROFILE)"; \
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
