#!/bin/zsh
set -euo pipefail

ENV_FILE="${HOME}/.omnirouter.env"
if [[ -f "${ENV_FILE}" ]]; then
  # shellcheck disable=SC1090
  source "${ENV_FILE}"
fi

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
APP_BIN="${REPO_DIR}/omnirouter"

if [[ ! -x "${APP_BIN}" ]]; then
  echo "omnirouter binary not found or not executable at: ${APP_BIN}" >&2
  echo "Build with: CGO_LDFLAGS="-framework UniformTypeIdentifiers" go build -tags production -o omnirouter" >&2
  exit 1
fi

exec "${APP_BIN}"
