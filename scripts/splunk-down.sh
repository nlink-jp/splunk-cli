#!/usr/bin/env bash
# Stop and remove the Splunk integration-test container.
set -euo pipefail

CONTAINER_NAME="splunk-test"

if podman container exists "$CONTAINER_NAME" 2>/dev/null; then
    podman rm -f "$CONTAINER_NAME" >/dev/null
    printf '[splunk-down] Container "%s" removed.\n' "$CONTAINER_NAME" >&2
else
    printf '[splunk-down] Container "%s" not found — nothing to do.\n' "$CONTAINER_NAME" >&2
fi
