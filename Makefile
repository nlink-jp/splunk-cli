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

# darwin ships arm64 only (no amd64, no universal/lipo). linux/windows keep their matrix.
PLATFORMS := darwin/arm64 linux/amd64 linux/arm64 windows/amd64

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
	@for p in $(PLATFORMS); do os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = windows ] && ext=".exe"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o dist/$(BINARY)-$$os-$$arch$$ext $(CMD) ; \
	done
	@scripts/codesign-darwin.sh dist/$(BINARY)-darwin-arm64 "$(CODESIGN_IDENTITY)" "$(BINARY)"

## package: Build all platforms, archive with version suffix (zip for
## darwin/windows, tar.gz for linux), bundle the canonical binary +
## README.md + LICENSE, and notarize the darwin build → dist/. Asset
## naming follows the org Release Archive Standard
## (splunk-cli-vX.Y.Z-<os>-<arch>.<ext>).
package: build-all
	@cd dist && for p in $(PLATFORMS); do os=$${p%/*}; arch=$${p#*/}; \
		ext=""; [ "$$os" = windows ] && ext=".exe"; \
		stage=_pkg; rm -rf $$stage; mkdir -p $$stage; \
		cp "$(BINARY)-$$os-$$arch$$ext" "$$stage/$(BINARY)$$ext"; \
		cp ../README.md ../LICENSE $$stage/; \
		base="$(BINARY)-$(VERSION)-$$os-$$arch"; \
		if [ "$$os" = linux ]; then ( cd $$stage && tar -czf "../$$base.tar.gz" * ); \
		else ( cd $$stage && zip -q "../$$base.zip" * ); fi; \
		rm -rf $$stage; \
	done
	@scripts/notarize-darwin.sh dist/$(BINARY)-$(VERSION)-darwin-arm64.zip "$(NOTARY_PROFILE)"

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
