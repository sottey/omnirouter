# OmniRouter Codex Prompt

Build a very small macOS-first desktop app named **OmniRouter**.

## Goal

OmniRouter lets the user type a prompt once, choose an LLM app from a dropdown, and send that prompt to the selected app.

The first MVP should support installed macOS apps like ChatGPT, Claude, and Gemini if available.

## Core Behavior

1. Show a simple UI with:
   - Dropdown/select box of configured LLM targets
   - Multi-line text box for the prompt
   - Send button
   - Optional status message area

2. Load targets from a JSON config file.

3. When the user clicks Send or presses Cmd+Enter:
   - Read selected target
   - Launch/focus the target macOS app
   - Copy the prompt text to the clipboard
   - Paste it into the target app
   - Press Enter to submit

## Tech Requirements

- Language: Go
- Use a simple desktop UI framework. Prefer **Wails** unless there is a clear reason not to.
- Config format: JSON
- Target platform for MVP: macOS
- Keep the project simple and readable.
- Do not add APIs, login handling, chat history, model selection, or response scraping.
- Do not refactor beyond what is needed for the MVP.

## Suggested Config

```json
{
  "targets": [
    {
      "name": "ChatGPT",
      "type": "mac_app",
      "app": "ChatGPT",
      "sendMode": "paste_enter",
      "startupDelayMs": 900
    },
    {
      "name": "Claude",
      "type": "mac_app",
      "app": "Claude",
      "sendMode": "paste_enter",
      "startupDelayMs": 900
    },
    {
      "name": "Gemini",
      "type": "mac_app",
      "app": "Gemini",
      "sendMode": "paste_enter",
      "startupDelayMs": 900
    }
  ]
}
```

## Suggested Go Structs

```go
type Config struct {
    Targets []Target `json:"targets"`
}

type Target struct {
    Name           string `json:"name"`
    Type           string `json:"type"`
    App            string `json:"app,omitempty"`
    URL            string `json:"url,omitempty"`
    SendMode       string `json:"sendMode"`
    StartupDelayMs int    `json:"startupDelayMs"`
}
```

## macOS Automation Details

For a `mac_app` target, use:

```bash
open -a "App Name"
```

Then wait for `startupDelayMs`.

To paste and submit:

```bash
printf '%s' "$PROMPT" | pbcopy
osascript -e 'tell application "System Events" to keystroke "v" using command down'
osascript -e 'tell application "System Events" to key code 36'
```

The app will need macOS Accessibility permissions for keystroke automation.

## Implementation Requirements

Create the project with a clean structure, for example:

```
omnirouter/
  main.go
  config.go
  automation.go
  types.go
  config.json
  frontend/
```

Implement:

- Config loading from `config.json`
- Basic validation:
  - target name required
  - target type required
  - app required for `mac_app`
  - sendMode defaults to `paste_enter`
  - startupDelayMs defaults to 900
- UI loads targets from config
- Send button calls backend send function
- Cmd+Enter submits the prompt
- Status/error messages shown in the UI
- Clear prompt after successful send only if simple to implement; otherwise leave text intact

## Important Constraints

- Keep this as an MVP.
- Do not implement provider APIs.
- Do not store prompt history.
- Do not scrape responses.
- Do not add account/auth features.
- Do not over-engineer plugin architecture yet.
- Prefer working, boring code over clever abstractions.

## Deliverables

Please generate the initial working project files.

Include:

1. Full source code
2. `config.json`
3. README with:
   - setup instructions
   - how to run
   - macOS Accessibility permission note
   - how to edit targets
   - current MVP limitations

## Acceptance Criteria

The MVP is successful when:

1. App starts.
2. Dropdown shows configured targets.
3. User can type a prompt.
4. User can send prompt to ChatGPT desktop app.
5. App launches/focuses ChatGPT if needed.
6. Prompt is pasted.
7. Enter is pressed automatically.
