# OmniRouter

OmniRouter is a small macOS-first desktop app that lets you type a prompt once, select a configured LLM target, and send the prompt to that app.

## Requirements

- Go 1.22+
- Wails CLI installed
- macOS

## Setup

1. Install the Wails CLI if you do not already have it:

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@latest
```

2. Install project dependencies:

```bash
go mod tidy
```

3. Start the app in development mode:

```bash
wails dev
```

4. Build the app with Wails:

```bash
wails build
```

5. If you want to build a direct runnable binary with `go build`, you must include the Wails production tag:

```bash
go build -tags production -o omnirouter
./omnirouter
```

Running a binary produced by plain `go build` without `-tags production` will panic with:

```text
Wails applications will not build without the correct build tags.
```

## macOS Accessibility Permission

This MVP uses `osascript` to paste and press Enter in another app. The built app, Terminal, or the process running OmniRouter will need macOS Accessibility permissions for keystroke automation to work.

## Menu Bar And Hotkey

On macOS, OmniRouter adds a menu bar item with `Show OmniRouter`, `Toggle Window`, `Settings`, `Reload Config`, and `Quit OmniRouter`.

The default global hotkey is `Cmd+Option+Space`, which toggles the main window.

## Edit Targets

Targets are defined in `config.json`.

Top-level window startup control:

```json
{
  "showWindowOnStartup": true,
  "defaultTarget": "Auto"
}
```

- `defaultTarget` is optional.
- If set, the dropdown selects that target on startup.
- It must match a target `name` exactly.

Window size can also be stored in `config.json`:

```json
{
  "windowWidth": 900,
  "windowHeight": 640
}
```

Window position can also be stored in `config.json`:

```json
{
  "windowX": 120,
  "windowY": 80
}
```

Optional send and hotkey behavior:

```json
{
  "autoHideAfterSend": false,
  "hotkeyMode": "toggle",
  "launcherWindowWidth": 760,
  "launcherWindowHeight": 280
}
```

- `hotkeyMode: "toggle"` uses the global hotkey to show or hide the normal window
- `hotkeyMode: "launcher"` uses the global hotkey to open a compact prompt-focused launcher window
- `autoHideAfterSend: true` hides the window after a successful send

OpenAI router configuration for an explicit `Auto` target:

```json
{
  "router": {
    "provider": "openai",
    "model": "gpt-4o-mini",
    "apiKeyEnv": "OPENAI_API_KEY"
  }
}
```

The OpenAI API key is loaded from the environment variable named by `router.apiKeyEnv`.

Each target currently supports:

```json
{
  "name": "ChatGPT",
  "type": "mac_app",
  "app": "ChatGPT",
  "shortcut": "Ctrl+1",
  "sendMode": "paste_enter",
  "startupDelayMs": 900
}
```

If `shortcut` is omitted, the first 9 targets still default to `Ctrl+1` through `Ctrl+9`.

Supported `sendMode` values:
- `paste_enter`: paste prompt and press Enter
- `paste_only`: paste prompt without pressing Enter

An explicit auto-routing target looks like this:

```json
{
  "name": "Auto",
  "type": "auto",
  "description": "Choose the best target for the prompt automatically."
}
```

Per-target descriptions are optional, but they improve routing quality:

```json
{
  "name": "Claude",
  "type": "mac_app",
  "app": "Claude",
  "description": "Good for careful reasoning, editing, and long-form analysis."
}
```

## Current MVP Limitations

- macOS only
- `mac_app` targets only
- No provider APIs
- No auth or account handling
- No prompt history
- No response scraping
- No model selection

## Run Automatically At Login

If you run OmniRouter outside your interactive shell, your `OPENAI_API_KEY` export is not available automatically.  
Use a user env file + launch agent:

1. Create `~/.omnirouter.env`:

```bash
cat > ~/.omnirouter.env <<'EOF'
export OPENAI_API_KEY="your_api_key_here"
EOF
chmod 600 ~/.omnirouter.env
```

2. Build the runnable binary:

```bash
go build -tags production -o omnirouter
```

3. Install login auto-start:

```bash
chmod +x scripts/install-launch-agent.sh
./scripts/install-launch-agent.sh
```

This installs `~/Library/LaunchAgents/com.omnirouter.plist` and starts OmniRouter at login using `scripts/run-omnirouter.sh`.
