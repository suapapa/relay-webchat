#!/bin/bash
set -euo pipefail

cd "$(dirname "$0")"
set -a
# shellcheck disable=SC1091
[ -f ../.env ] && source ../.env
[ -f .env ] && source .env
set +a

go build
./processor -ws "${RELAY_WS:-wss://homin.dev/webchat-relay/ws}" "$@"
