#!/usr/bin/env bash
# Start a Splunk container for integration testing.
# Usage: scripts/splunk-up.sh [splunk-image-tag]
# Default image tag: 9.4
#
# Exports (for use in the calling shell via eval):
#   SPLUNK_HOST   https://localhost:<mapped-port>
#   SPLUNK_TOKEN  admin session token
set -euo pipefail

CONTAINER_NAME="splunk-test"
IMAGE_TAG="${1:-9.4}"
IMAGE="docker.io/splunk/splunk:${IMAGE_TAG}"
SPLUNK_PASSWORD="Admin1234!"
READY_TIMEOUT=300  # seconds (initial provisioning can take ~2 min on first run)

# ── helpers ────────────────────────────────────────────────────────────────────

log() { printf '[splunk-up] %s\n' "$*" >&2; }
die() { printf '[splunk-up] ERROR: %s\n' "$*" >&2; exit 1; }

# ── check for existing container ───────────────────────────────────────────────

if podman container exists "$CONTAINER_NAME" 2>/dev/null; then
    STATE=$(podman inspect --format '{{.State.Status}}' "$CONTAINER_NAME")
    if [[ "$STATE" == "running" ]]; then
        log "Container '$CONTAINER_NAME' is already running — skipping start."
    else
        log "Removing stopped container '$CONTAINER_NAME'."
        podman rm "$CONTAINER_NAME" >/dev/null
        STATE=""
    fi
fi

if ! podman container exists "$CONTAINER_NAME" 2>/dev/null; then
    # Pick a random available port in 18000-18999
    PORT=$(python3 -c "
import socket, random
for _ in range(100):
    p = random.randint(18000, 18999)
    with socket.socket() as s:
        if s.connect_ex(('127.0.0.1', p)) != 0:
            print(p); break
")
    [[ -n "$PORT" ]] || die "Could not find a free port."

    log "Pulling image ${IMAGE} (may take a while on first run)..."
    podman pull --platform linux/amd64 "$IMAGE" >&2

    log "Starting Splunk on localhost:${PORT}..."
    podman run -d \
        --name "$CONTAINER_NAME" \
        --platform linux/amd64 \
        -p "${PORT}:8089" \
        -e SPLUNK_START_ARGS="--accept-license" \
        -e SPLUNK_PASSWORD="$SPLUNK_PASSWORD" \
        "$IMAGE" >/dev/null
fi

# Resolve the mapped port (handles the "already running" case too)
PORT=$(podman port "$CONTAINER_NAME" 8089/tcp | cut -d: -f2)
HOST="https://localhost:${PORT}"

# ── wait for REST API to be ready ──────────────────────────────────────────────

log "Waiting for Splunk REST API at ${HOST} (up to ${READY_TIMEOUT}s)..."
deadline=$(( $(date +%s) + READY_TIMEOUT ))
until curl -sk -u "admin:${SPLUNK_PASSWORD}" \
        "${HOST}/services/server/info?output_mode=json" \
        -o /dev/null -w '%{http_code}' 2>/dev/null | grep -q '^200$'; do
    if (( $(date +%s) >= deadline )); then
        podman logs "$CONTAINER_NAME" >&2
        die "Splunk did not become ready within ${READY_TIMEOUT}s."
    fi
    sleep 3
done
log "Splunk is ready."

# ── obtain a session token ─────────────────────────────────────────────────────

TOKEN=$(curl -sk \
    -d "username=admin&password=${SPLUNK_PASSWORD}&output_mode=json" \
    "${HOST}/services/auth/login" \
    | python3 -c "import sys,json; print(json.load(sys.stdin)['sessionKey'])")

[[ -n "$TOKEN" ]] || die "Failed to obtain session token."
log "Session token obtained."

# ── emit export statements ─────────────────────────────────────────────────────

printf 'export SPLUNK_HOST="%s"\n'  "$HOST"
printf 'export SPLUNK_TOKEN="%s"\n' "$TOKEN"
log "Run:  eval \"\$(scripts/splunk-up.sh)\"  to set env vars in your shell."
