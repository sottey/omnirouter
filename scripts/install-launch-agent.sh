#!/bin/zsh
set -euo pipefail

REPO_DIR="$(cd "$(dirname "$0")/.." && pwd)"
RUN_SCRIPT="${REPO_DIR}/scripts/run-omnirouter.sh"
AGENT_ID="com.omnirouter"
PLIST_PATH="${HOME}/Library/LaunchAgents/${AGENT_ID}.plist"

chmod +x "${RUN_SCRIPT}"

cat > "${PLIST_PATH}" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>${AGENT_ID}</string>
  <key>ProgramArguments</key>
  <array>
    <string>${RUN_SCRIPT}</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
  <key>KeepAlive</key>
  <false/>
  <key>WorkingDirectory</key>
  <string>${REPO_DIR}</string>
  <key>StandardOutPath</key>
  <string>${REPO_DIR}/app.log</string>
  <key>StandardErrorPath</key>
  <string>${REPO_DIR}/app.log</string>
</dict>
</plist>
PLIST

launchctl unload "${PLIST_PATH}" >/dev/null 2>&1 || true
launchctl load "${PLIST_PATH}"
echo "Installed and loaded ${PLIST_PATH}"
