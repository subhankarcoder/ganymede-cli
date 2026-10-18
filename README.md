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

## 📦 Installation

### 1. Via Windows Package Manager (WinGet - Recommended)
Install with a single command in any Windows terminal (PowerShell, Command Prompt, or Windows Terminal):

```powershell
# Install directly by short name
winget install ganymede

# Or via full package identifier
winget install ebsglobal.ganymede
```

> **Note**: WinGet automatically exposes the `ganymede` command to your system `PATH`. You can run `ganymede` immediately from any terminal!

### 2. Manual / Build from Source
```powershell
# Clone the repository
git clone https://github.com/subhankarcoder/ganymede-cli.git
cd ganymede-cli

# Build executable (Windows GUI subsystem enables silent background execution)
go build -trimpath -ldflags "-H=windowsgui -s -w" -o ganymede.exe .

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
| `ganymede version` / `--version` | Display current release version |
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

---

## 🚀 CI/CD & Automated WinGet Releases

Ganymede CLI uses GitHub Actions for continuous integration, release packaging, and automatic publishing to Microsoft's official Windows Package Manager repository (`microsoft/winget-pkgs`).

### How It Works:
1. **Pushing Changes**: Every push to `main` and pull request automatically triggers `.github/workflows/ci.yml` which vets and cross-compiles Windows AMD64 and ARM64 binaries.
2. **Releasing New Versions**: Pushing a version tag triggers `.github/workflows/release.yml`:
   ```powershell
   git tag v0.1.0
   git push origin v0.1.0
   ```
3. **Automated Pipeline**:
   - Cross-compiles hardened Windows binaries (`amd64` and `arm64`) with `-trimpath` and GUI subsystem flags.
   - Archives binaries with documentation into release zips.
   - Computes SHA256 checksums.
   - Creates a GitHub Release.
   - Uses `vedantmgoyal9/winget-releaser` to submit a pull request directly to `microsoft/winget-pkgs`.

---

## 📄 License

This project is licensed under the [MIT License](LICENSE).
