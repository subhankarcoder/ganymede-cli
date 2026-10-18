# ✦ Ganymede CLI

> **Precision Screen Eyedropper & Palette Engine for Designers and Developers**

Ganymede CLI lets designers pick colors from anywhere on their active screen with a single global hotkey (`Win + Alt + P`), an interactive Next.js-style setup wizard, live system hotkey conflict scanning, real-time loupe preview card, and instant clipboard copying in **HEX**, **RGB**, or **CMYK**.

---

## ✨ Features

- 🎯 **Figma/Canva-Style Precision Eyedropper**: An authentic design tool experience with a high-DPI angled pipette cursor whose nozzle points directly at `(1, 1)`.
- 🔍 **Circular Magnifying Loupe**: An authentic Figma/Canva circular lens with an 11×11 zoomed pixel grid, 1px separating grid lines, center pixel reticle, and an outer 7px color swatch ring filled with the live color.
- ⚡ **Zero-Lag Freeze-Frame Architecture**: Instantaneous 5ms screen snapshot in local RAM. Zero `GetDC(0)` polling during mouse movement, meaning 144Hz+ fluid movement with zero CPU spikes, zero stutter, and zero Windows freezes.
- 📋 **One-Click / Spacebar Clipboard Copying**: Left-click or press Spacebar/Enter to sample and copy the color code to your Windows clipboard. Press `Esc` or right-click to cancel.
- 🚀 **Next.js-Style Setup Wizard**: Interactive CLI prompts with keyboard buffer draining, arrow-key navigation, radio indicators (`❯ ●`), live scanning indicators, and clean inline completion badges.
- ⚡ **Live Hotkey Conflict Detection**: Scans Windows hotkey registrations in real time to alert you if a shortcut is already locked by another program and guides you to conflict-free options.
- 🔄 **Windows Startup Integration**: Option to silently activate Ganymede on Windows login so your hotkey is always available.
- 🛠️ **Handy Configuration Commands**: Quickly toggle or change color modes, hotkeys, startup status, or themes anytime.

---

## 📦 Quick Start

### 1. Build & Run
```powershell
# Build executable (targeting Windows GUI subsystem enables completely silent background startup)
go build -ldflags "-H=windowsgui" -o ganymede.exe .

# Run the interactive setup wizard
.\ganymede.exe init
```

### 2. Interactive Setup Flow
During installation, Ganymede walks you through 3 simple questions:

1. **Preferred Color Mode**:
   - `HEX` (`#6366F1`) - Modern web & UI design (CSS, HTML, Tailwind, Figma)
   - `RGB` (`rgb(99, 102, 241)`) - Digital graphics, WebGL, shaders, canvas
   - `CMYK` (`cmyk(59%, 58%, 0%, 5%)`) - Print media & prepress production

2. **Global Shortcut Pair (with Live Conflict Scanning)**:
   - `Win + Alt + P` `[✓ Available]` *(Recommended for designers: P for Palette/Pixel)*
   - `Ctrl + Alt + C` `[✓ Available]` *(Classic: C for Color)*
   - `Alt + Shift + X` `[✓ Available]` *(Ergonomic: X for eXtract)*
   *(If another application is using a combination, Ganymede flags it in red `[✗ In use]` and guides you).*

3. **Activate on Windows Startup**:
   - `Yes (Recommended)` - Automatically runs the listener in the background on Windows login.
   - `No` - Manual start required (`ganymede start`).

---

## 🎮 Command Reference

| Command | Description |
| :--- | :--- |
| `ganymede init` / `setup` | Launch the 3-step interactive setup wizard |
| `ganymede pick` | Immediately trigger the screen eyedropper from your terminal |
| `ganymede start` / `listen` | Start the background hotkey listener |
| `ganymede status` | Display active color mode, hotkey status, and startup health |
| `ganymede config` | Open interactive menu to adjust any of your primary inputs |
| `ganymede config mode <hex\|rgb\|cmyk>` | Switch clipboard format directly |
| `ganymede config hotkey` | Re-test system shortcuts and select a new hotkey |
| `ganymede config startup <enable\|disable>` | Toggle automatic launch on Windows startup |
| `ganymede config theme <name>` | Change palette theme (`ganymede`, `rainbow`, `cyberpunk`, `sunset`, `aurora`) |
| `ganymede help` / `--help` | Display comprehensive manual and usage guide |

---

## ⚙️ Configuration File

Preferences are stored in JSON format at:
```
%APPDATA%\ganymede\config.json
```
Example:
```json
{
  "color_mode": "hex",
  "hotkey": "win+alt+p",
  "startup": true,
  "theme": "ganymede",
  "sound": true
}
```
