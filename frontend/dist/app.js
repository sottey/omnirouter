async function getAppBindings() {
  if (!window.go || !window.go.main || !window.go.main.App) {
    throw new Error("Wails bindings are not available");
  }

  return window.go.main.App;
}

const targetSelect = document.getElementById("targetSelect");
const promptInput = document.getElementById("promptInput");
const sendButton = document.getElementById("sendButton");
const settingsButton = document.getElementById("settingsButton");
const statusMessage = document.getElementById("statusMessage");
const shortcutLegend = document.getElementById("shortcutLegend");
const settingsModal = document.getElementById("settingsModal");
const settingsDefaultTarget = document.getElementById("settingsDefaultTarget");
const settingsTargetSelect = document.getElementById("settingsTargetSelect");
const settingsTargetSendMode = document.getElementById("settingsTargetSendMode");
const settingsTestButton = document.getElementById("settingsTestButton");
const settingsHotkeyMode = document.getElementById("settingsHotkeyMode");
const settingsShowOnStartup = document.getElementById("settingsShowOnStartup");
const settingsAutoHideAfterSend = document.getElementById("settingsAutoHideAfterSend");
const settingsLauncherWidth = document.getElementById("settingsLauncherWidth");
const settingsLauncherHeight = document.getElementById("settingsLauncherHeight");
const settingsCancelButton = document.getElementById("settingsCancelButton");
const settingsSaveButton = document.getElementById("settingsSaveButton");
let currentConfig = null;
let shortcutMap = {};
let resizeSaveTimeoutId = null;
let windowStateSaveIntervalId = null;
let configRefreshIntervalId = null;
let settingsOpen = false;
let settingsSendModeByTarget = {};

function setStatus(message, tone) {
  statusMessage.textContent = message;
  statusMessage.className = "status";
  if (tone) {
    statusMessage.classList.add(tone);
  }
}

function setSendingState(isSending) {
  sendButton.disabled = isSending;
  sendButton.textContent = isSending ? "Sending..." : "Send";
}

function closeSettingsModal() {
  settingsModal.classList.add("hidden");
  settingsOpen = false;
  promptInput.focus();
}

function openSettingsModal() {
  if (!currentConfig) {
    setStatus("Config not loaded.", "error");
    return;
  }

  const targets = currentConfig.targets || [];
  const editableTargets = targets.filter((target) => target.type === "mac_app");
  settingsDefaultTarget.innerHTML = "";
  settingsTargetSelect.innerHTML = "";
  settingsSendModeByTarget = {};

  const emptyOption = document.createElement("option");
  emptyOption.value = "";
  emptyOption.textContent = "(none)";
  settingsDefaultTarget.appendChild(emptyOption);

  for (const target of targets) {
    settingsSendModeByTarget[target.name] = (target.sendMode || "paste_enter").trim().toLowerCase();

    const option = document.createElement("option");
    option.value = target.name;
    option.textContent = target.name;
    settingsDefaultTarget.appendChild(option);
  }

  for (const target of editableTargets) {
    const option = document.createElement("option");
    option.value = target.name;
    option.textContent = target.name;
    settingsTargetSelect.appendChild(option);
  }

  settingsDefaultTarget.value = currentConfig.defaultTarget || "";
  settingsHotkeyMode.value = currentConfig.hotkeyMode === "launcher" ? "launcher" : "toggle";
  settingsShowOnStartup.checked = currentConfig.showWindowOnStartup !== false;
  settingsAutoHideAfterSend.checked = currentConfig.autoHideAfterSend === true;
  settingsLauncherWidth.value = String(currentConfig.launcherWindowWidth || 760);
  settingsLauncherHeight.value = String(currentConfig.launcherWindowHeight || 280);
  if (editableTargets.length > 0) {
    settingsTargetSelect.value = editableTargets[0].name;
    settingsTargetSendMode.value = settingsSendModeByTarget[editableTargets[0].name] || "paste_enter";
    settingsTargetSelect.disabled = false;
    settingsTargetSendMode.disabled = false;
    settingsTestButton.disabled = false;
  } else {
    settingsTargetSendMode.value = "paste_enter";
    settingsTargetSelect.disabled = true;
    settingsTargetSendMode.disabled = true;
    settingsTestButton.disabled = true;
  }

  settingsModal.classList.remove("hidden");
  settingsOpen = true;
}

async function saveSettings() {
  if (!currentConfig) {
    setStatus("Config not loaded.", "error");
    return;
  }

  const launcherWidth = Number.parseInt(settingsLauncherWidth.value, 10);
  const launcherHeight = Number.parseInt(settingsLauncherHeight.value, 10);
  if (!Number.isFinite(launcherWidth) || launcherWidth <= 0) {
    setStatus("Launcher width must be a positive number.", "error");
    return;
  }
  if (!Number.isFinite(launcherHeight) || launcherHeight <= 0) {
    setStatus("Launcher height must be a positive number.", "error");
    return;
  }

  if (!settingsTargetSendMode.value) {
    setStatus("Select a send mode.", "error");
    return;
  }

  settingsSendModeByTarget[settingsTargetSelect.value] = settingsTargetSendMode.value;
  const nextTargets = (currentConfig.targets || []).map((target) => {
    if (!settingsSendModeByTarget[target.name]) {
      return target;
    }

    return {
      ...target,
      sendMode: settingsSendModeByTarget[target.name],
    };
  });

  const nextConfig = {
    ...currentConfig,
    defaultTarget: settingsDefaultTarget.value.trim(),
    hotkeyMode: settingsHotkeyMode.value,
    showWindowOnStartup: settingsShowOnStartup.checked,
    autoHideAfterSend: settingsAutoHideAfterSend.checked,
    launcherWindowWidth: launcherWidth,
    launcherWindowHeight: launcherHeight,
    targets: nextTargets,
  };

  settingsSaveButton.disabled = true;
  settingsSaveButton.textContent = "Saving...";
  try {
    const bindings = await getAppBindings();
    await bindings.SaveConfig(nextConfig);
    await loadConfig({ silent: true });
    closeSettingsModal();
    setStatus("Settings saved.", "success");
  } catch (error) {
    setStatus(error.message || String(error), "error");
  } finally {
    settingsSaveButton.disabled = false;
    settingsSaveButton.textContent = "Save";
  }
}

async function testSelectedTarget() {
  const targetName = settingsTargetSelect.value.trim();
  if (!targetName) {
    setStatus("Select a target to test.", "error");
    return;
  }

  try {
    const bindings = await getAppBindings();
    settingsTestButton.disabled = true;
    settingsTestButton.textContent = "Testing...";
    await bindings.TestTarget(targetName);
    setStatus(`Test sent to ${targetName}.`, "success");
  } catch (error) {
    setStatus(error.message || String(error), "error");
  } finally {
    settingsTestButton.disabled = settingsTargetSelect.disabled;
    settingsTestButton.textContent = "Test Target";
  }
}

function selectTargetByValue(targetName) {
  if (!targetName) {
    return;
  }

  const targetIndex = Array.from(targetSelect.options).findIndex(
    (option) => option.value === targetName,
  );
  if (targetIndex === -1) {
    return;
  }

  const selectionStart = promptInput.selectionStart;
  const selectionEnd = promptInput.selectionEnd;

  targetSelect.selectedIndex = targetIndex;
  promptInput.focus();
  if (selectionStart !== null && selectionEnd !== null) {
    promptInput.setSelectionRange(selectionStart, selectionEnd);
  }
}

function normaliseShortcut(shortcut) {
  if (!shortcut) {
    return "";
  }

  return shortcut.trim().toLowerCase();
}

function buildShortcutMap(targets) {
  const nextShortcutMap = {};

  for (let index = 0; index < targets.length; index += 1) {
    const target = targets[index];
    const configuredShortcut = normaliseShortcut(target.shortcut);

    if (configuredShortcut) {
      nextShortcutMap[configuredShortcut] = target.name;
      continue;
    }

    if (index < 9) {
      nextShortcutMap[`ctrl+${index + 1}`] = target.name;
    }
  }

  shortcutMap = nextShortcutMap;
}

function getShortcutLabel(target, index) {
  if (target.shortcut && target.shortcut.trim()) {
    return target.shortcut.trim();
  }

  if (index < 9) {
    return `Ctrl+${index + 1}`;
  }

  return "";
}

function getShortcutTargets(targets) {
  return targets
    .map((target, index) => ({
      name: target.name,
      shortcut: getShortcutLabel(target, index),
    }))
    .filter((target) => target.shortcut);
}

function renderShortcutLegend(targets) {
  const shortcutTargets = getShortcutTargets(targets);

  if (shortcutTargets.length === 0) {
    shortcutLegend.textContent = "";
    return;
  }

  shortcutLegend.innerHTML = shortcutTargets
    .map(
      (target) =>
        `<button type="button" class="shortcut-chip${
          target.name === targetSelect.value ? " shortcut-chip-active" : ""
        }" data-target-name="${target.name}">${target.shortcut}: ${target.name}</button>`,
    )
    .join("");
}

async function loadConfig(options = {}) {
  const { silent = false } = options;

  if (!silent) {
    setStatus("Loading targets...");
  }

  try {
    const bindings = await getAppBindings();
    const config = await bindings.GetConfig();
    const previousTarget = targetSelect.value;
    const targets = config.targets || [];

    currentConfig = config;
    buildShortcutMap(targets);

    targetSelect.innerHTML = "";

    for (const target of targets) {
      const option = document.createElement("option");
      option.value = target.name;
      option.textContent = target.name;
      targetSelect.appendChild(option);
    }

    if (previousTarget && targets.some((target) => target.name === previousTarget)) {
      targetSelect.value = previousTarget;
    } else if (
      config.defaultTarget &&
      targets.some((target) => target.name === config.defaultTarget)
    ) {
      targetSelect.value = config.defaultTarget;
    }

    renderShortcutLegend(targets);
    if (!silent) {
      setStatus("");
    }
  } catch (error) {
    setStatus(error.message || String(error), "error");
  }
}

async function sendPrompt() {
  const targetName = targetSelect.value;
  const prompt = promptInput.value;

  if (!targetName) {
    setStatus("Select a target.", "error");
    return;
  }

  if (!prompt.trim()) {
    setStatus("Prompt is required.", "error");
    return;
  }

  setSendingState(true);
  setStatus("Sending prompt...");

  try {
    const bindings = await getAppBindings();
    const chosenTargetName = await bindings.SendPrompt(targetName, prompt);
    promptInput.value = "";
    promptInput.focus();
    if (currentConfig && currentConfig.autoHideAfterSend) {
      await bindings.HideMainWindow();
    }
    if (targetName === chosenTargetName) {
      setStatus(`Sent to ${chosenTargetName}.`, "success");
    } else {
      setStatus(`Auto selected ${chosenTargetName}. Sent successfully.`, "success");
    }
  } catch (error) {
    setStatus(error.message || String(error), "error");
  } finally {
    setSendingState(false);
  }
}

async function saveWindowSize() {
  try {
    const bindings = await getAppBindings();
    await bindings.SaveWindowState();
  } catch (error) {
    console.error("Failed to save window size:", error);
  }
}

async function maybeOpenSettingsFromMenu() {
  try {
    const bindings = await getAppBindings();
    const shouldOpenSettings = await bindings.ConsumeOpenSettingsRequest();
    if (!shouldOpenSettings) {
      return;
    }

    if (!currentConfig) {
      await loadConfig({ silent: true });
    }
    openSettingsModal();
  } catch (error) {
    console.error("Failed to handle settings open request:", error);
  }
}

sendButton.addEventListener("click", () => {
  void sendPrompt();
});

settingsButton.addEventListener("click", () => {
  openSettingsModal();
});

settingsCancelButton.addEventListener("click", () => {
  closeSettingsModal();
});

settingsSaveButton.addEventListener("click", () => {
  void saveSettings();
});

settingsTargetSelect.addEventListener("change", () => {
  const targetName = settingsTargetSelect.value;
  settingsTargetSendMode.value = settingsSendModeByTarget[targetName] || "paste_enter";
});

settingsTargetSendMode.addEventListener("change", () => {
  const targetName = settingsTargetSelect.value;
  if (!targetName) {
    return;
  }
  settingsSendModeByTarget[targetName] = settingsTargetSendMode.value;
});

targetSelect.addEventListener("change", () => {
  if (!currentConfig) {
    return;
  }

  renderShortcutLegend(currentConfig.targets || []);
});

shortcutLegend.addEventListener("click", (event) => {
  const chip = event.target.closest("[data-target-name]");
  if (!chip) {
    return;
  }

  selectTargetByValue(chip.dataset.targetName);
  if (!currentConfig) {
    return;
  }

  renderShortcutLegend(currentConfig.targets || []);
});

settingsTestButton.addEventListener("click", () => {
  void testSelectedTarget();
});

promptInput.addEventListener("keydown", (event) => {
  if (event.key === "Enter" && event.metaKey) {
    event.preventDefault();
    void sendPrompt();
  }
});

window.addEventListener("keydown", (event) => {
  if (event.key === "Escape") {
    event.preventDefault();
    if (settingsOpen) {
      closeSettingsModal();
      return;
    }
    void getAppBindings()
      .then((bindings) => bindings.HideMainWindow())
      .catch((error) => {
        console.error("Failed to hide window:", error);
      });
    return;
  }

  if (settingsOpen) {
    return;
  }

  if (!event.ctrlKey || event.metaKey || event.altKey || event.shiftKey) {
    return;
  }

  const shortcutKey = normaliseShortcut(`Ctrl+${event.key}`);
  const targetName = shortcutMap[shortcutKey];
  if (!targetName) {
    return;
  }

  event.preventDefault();
  selectTargetByValue(targetName);
});

window.addEventListener("resize", () => {
  if (resizeSaveTimeoutId !== null) {
    window.clearTimeout(resizeSaveTimeoutId);
  }

  resizeSaveTimeoutId = window.setTimeout(() => {
    resizeSaveTimeoutId = null;
    void saveWindowSize();
  }, 250);
});

window.addEventListener("focus", () => {
  if (!settingsOpen) {
    promptInput.focus();
  }
});

if (windowStateSaveIntervalId === null) {
  windowStateSaveIntervalId = window.setInterval(() => {
    void saveWindowSize();
  }, 1000);
}

if (configRefreshIntervalId === null) {
  configRefreshIntervalId = window.setInterval(() => {
    void loadConfig({ silent: true });
    void maybeOpenSettingsFromMenu();
  }, 1000);
}

void loadConfig();
void maybeOpenSettingsFromMenu();
