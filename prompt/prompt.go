package prompt

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"ganymede-cli/banner"
	"ganymede-cli/config"
	"ganymede-cli/hotkey"
	"ganymede-cli/startup"
)

var (
	msvcrt             = syscall.NewLazyDLL("msvcrt.dll")
	getchProc          = msvcrt.NewProc("_getch")
	kbhitProc          = msvcrt.NewProc("_kbhit")
	kernel32           = syscall.NewLazyDLL("kernel32.dll")
	getConsoleModeProc = kernel32.NewProc("GetConsoleMode")
	setConsoleModeProc = kernel32.NewProc("SetConsoleMode")
	getStdHandleProc   = kernel32.NewProc("GetStdHandle")
	attachConsoleProc  = kernel32.NewProc("AttachConsole")
)

const (
	attachParentProcess = ^uintptr(0)        // 0xFFFFFFFF
	stdInputHandle      = uint32(0xFFFFFFF6) // STD_INPUT_HANDLE (-10)
	stdOutputHandle     = uint32(0xFFFFFFF5) // STD_OUTPUT_HANDLE (-11)
)

func init() {
	enableVTProcessing()
}

func enableVTProcessing() {
	// If launched as windowsgui from an interactive console, attach
	attachConsoleProc.Call(attachParentProcess)

	var mode uint32
	hOut, _, _ := getStdHandleProc.Call(uintptr(stdOutputHandle))
	if hOut == 0 || hOut == uintptr(syscall.InvalidHandle) {
		hOut = uintptr(syscall.Stdout)
	}
	r, _, _ := getConsoleModeProc.Call(hOut, uintptr(unsafe.Pointer(&mode)))
	if r != 0 {
		const enableVT = 0x0004
		setConsoleModeProc.Call(hOut, uintptr(mode|enableVT))
	}
}

// Option represents a selectable item in Next.js style prompts
type Option struct {
	ID               string
	Label            string
	Badge            string
	Description      string
	Available        bool
	ShowAvailability bool
	Warning          string
}

// isInteractive checks if stdin is attached to an interactive Windows console
func isInteractive() bool {
	fileInfo, err := os.Stdin.Stat()
	if err == nil && (fileInfo.Mode()&os.ModeCharDevice) != 0 {
		return true
	}
	// Check the Windows console input buffer handle (critical for -H=windowsgui subsystem)
	hIn, _, _ := getStdHandleProc.Call(uintptr(stdInputHandle))
	if hIn != 0 && hIn != uintptr(syscall.InvalidHandle) {
		var mode uint32
		r, _, _ := getConsoleModeProc.Call(hIn, uintptr(unsafe.Pointer(&mode)))
		return r != 0
	}
	return false
}

func drainKeyboardBuffer() {
	time.Sleep(20 * time.Millisecond)
	for {
		r, _, _ := kbhitProc.Call()
		if r == 0 {
			break
		}
		getchProc.Call()
	}
}

// readKey reads a single keypress without waiting for enter
func readKey() (int, int) {
	r, _, _ := getchProc.Call()
	code := int(r)
	if code == 0 || code == 224 {
		r2, _, _ := getchProc.Call()
		return 0, int(r2) // Extended key: 72=Up, 80=Down, 75=Left, 77=Right
	}
	return code, 0
}

// Select displays an interactive Next.js style radio selection prompt
func Select(question string, options []Option, defaultIdx int) (int, error) {
	if defaultIdx < 0 || defaultIdx >= len(options) {
		defaultIdx = 0
	}

	drainKeyboardBuffer()

	// Non-interactive fallback
	if !isInteractive() {
		return fallbackSelect(question, options, defaultIdx)
	}

	selected := defaultIdx
	renderedLines := 0

	// Hide cursor during interactive menu
	fmt.Print("\x1b[?25l")
	defer fmt.Print("\x1b[?25h")

	render := func(firstRender bool) {
		if !firstRender && renderedLines > 0 {
			fmt.Printf("\r\x1b[%dA", renderedLines)
		}

		var out strings.Builder
		cyan := banner.Color{R: 56, G: 189, B: 248}.RGB()
		gray := banner.Color{R: 156, G: 163, B: 175}.RGB()

		// Question header
		out.WriteString(fmt.Sprintf("\x1b[2K%s%s?%s %s%s%s\n",
			banner.Bold(), cyan, banner.Reset(),
			banner.Bold(), question, banner.Reset(),
		))
		renderedLines = 1

		for i, opt := range options {
			var line strings.Builder
			isSelected := (i == selected)

			// Neon green for selected option
			neonGreen := banner.Color{R: 57, G: 255, B: 20}.RGB()

			if isSelected {
				line.WriteString(fmt.Sprintf("  %s%s❯ ●%s ", banner.Bold(), neonGreen, banner.Reset()))
				line.WriteString(fmt.Sprintf("%s%s%s%s", banner.Bold(), neonGreen, opt.Label, banner.Reset()))
			} else {
				line.WriteString(fmt.Sprintf("    %s○ %s%s", gray, opt.Label, banner.Reset()))
			}

			// Recommended badge if present (e.g. startup / recommended hotkey)
			if opt.Badge != "" {
				line.WriteString("  " + opt.Badge)
			}

			// Show availability tag ONLY on the active (selected) option
			if opt.ShowAvailability && isSelected {
				if opt.Available {
					// Distinct cyan/sky color for available tag (since selected text is neon green)
					availCol := banner.Color{R: 56, G: 189, B: 248}.RGB()
					line.WriteString(fmt.Sprintf("  %s[✓ Available]%s", availCol, banner.Reset()))
				} else {
					redCol := banner.Color{R: 239, G: 68, B: 68}.RGB()
					line.WriteString(fmt.Sprintf("  %s[✗ In use by system]%s", redCol, banner.Reset()))
				}
			}

			if opt.Description != "" {
				line.WriteString(fmt.Sprintf(" %s- %s%s", gray, opt.Description, banner.Reset()))
			}

			out.WriteString("\x1b[2K" + line.String() + "\n")
			renderedLines++
		}

		// Contextual hint
		curOpt := options[selected]
		var hint strings.Builder
		if curOpt.Warning != "" {
			yellow := banner.Color{R: 251, G: 191, B: 36}.RGB()
			hint.WriteString(fmt.Sprintf("    %s⚠ %s%s", yellow, curOpt.Warning, banner.Reset()))
		} else {
			hint.WriteString(fmt.Sprintf("    %s(Use arrow keys ↑/↓, Enter to confirm)%s", gray, banner.Reset()))
		}
		out.WriteString("\x1b[2K" + hint.String() + "\n")
		renderedLines++

		fmt.Print(out.String())
	}

	render(true)

	for {
		primary, ext := readKey()

		// Ctrl+C
		if primary == 3 {
			fmt.Print("\x1b[?25h\n")
			os.Exit(0)
		}

		// Enter (\r = 13, \n = 10)
		if primary == 13 || primary == 10 {
			drainKeyboardBuffer()
			break
		}

		// Up arrow or 'k'
		if ext == 72 || primary == 'k' || primary == 'K' {
			selected--
			if selected < 0 {
				selected = len(options) - 1
			}
			render(false)
		}

		// Down arrow or 'j'
		if ext == 80 || primary == 'j' || primary == 'J' {
			selected++
			if selected >= len(options) {
				selected = 0
			}
			render(false)
		}

		// Number shortcuts '1'..'9'
		if primary >= '1' && primary <= '9' {
			idx := primary - '1'
			if idx < len(options) {
				selected = idx
				render(false)
			}
		}
	}

	// Clear prompt and rewrite with clean completed state
	if renderedLines > 0 {
		fmt.Printf("\r\x1b[%dA", renderedLines)
		for i := 0; i < renderedLines; i++ {
			fmt.Print("\x1b[2K\n")
		}
		fmt.Printf("\r\x1b[%dA", renderedLines)
	}

	green := banner.Color{R: 52, G: 199, B: 89}.RGB()
	gray := banner.Color{R: 156, G: 163, B: 175}.RGB()
	fmt.Printf("\x1b[2K%s%s✔%s %s%s%s %s›%s %s%s%s\n",
		banner.Bold(), green, banner.Reset(),
		banner.Bold(), question, banner.Reset(),
		gray, banner.Reset(),
		banner.Bold(), options[selected].Label, banner.Reset(),
	)

	return selected, nil
}

func fallbackSelect(question string, options []Option, defaultIdx int) (int, error) {
	fmt.Printf("\n? %s\n", question)
	for i, opt := range options {
		desc := ""
		if opt.Description != "" {
			desc = " - " + opt.Description
		}
		fmt.Printf("  [%d] %s%s\n", i+1, opt.Label, desc)
	}
	fmt.Printf("Select [1-%d] (default %d): ", len(options), defaultIdx+1)

	reader := bufio.NewReader(os.Stdin)
	text, _ := reader.ReadString('\n')
	text = strings.TrimSpace(text)
	if text == "" {
		return defaultIdx, nil
	}

	var choice int
	_, err := fmt.Sscanf(text, "%d", &choice)
	if err == nil && choice >= 1 && choice <= len(options) {
		return choice - 1, nil
	}
	return defaultIdx, nil
}

// RunSetupWizard runs the complete 3-question setup flow
func RunSetupWizard(existingCfg *config.Config) (*config.Config, error) {
	cfg := config.DefaultConfig()
	if existingCfg != nil {
		cfg = existingCfg
	}

	fmt.Println()
	fmt.Printf("  %s%sWelcome to Ganymede. You need to choose 3 quick setup preferences to start the installation:%s\n\n",
		banner.Bold(), banner.Color{R: 251, G: 191, B: 36}.RGB(), banner.Reset())

	// -------------------------------------------------------------
	// QUESTION 1: Color Mode
	// -------------------------------------------------------------
	colorOptions := []Option{
		{
			ID:    "hex",
			Label: "HEX",
		},
		{
			ID:    "rgb",
			Label: "RGB",
		},
		{
			ID:    "cmyk",
			Label: "CMYK",
		},
	}

	defaultColorIdx := 0
	for i, opt := range colorOptions {
		if opt.ID == cfg.ColorMode {
			defaultColorIdx = i
			break
		}
	}

	colorIdx, err := Select("In which color mode do you want to copy the code to?", colorOptions, defaultColorIdx)
	if err != nil {
		return nil, err
	}
	cfg.ColorMode = colorOptions[colorIdx].ID

	// -------------------------------------------------------------
	// QUESTION 2: Hotkey Pairs with Live Conflict Checking
	// -------------------------------------------------------------
	fmt.Printf("  %s⠋ Scanning Windows hotkeys for system availability...%s",
		banner.Color{R: 147, G: 197, B: 253}.RGB(), banner.Reset())
	scannedPresets := hotkey.ScanPresetOptions()
	// Clear the scanning line
	fmt.Print("\r\x1b[2K")

	var hotkeyOptions []Option
	defaultHotkeyIdx := 0

	for i, opt := range scannedPresets {
		var badge string
		var warning string

		if opt.Key == "win+alt+p" {
			gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
			badge = fmt.Sprintf("%s(Recommended)%s", gold, banner.Reset())
		}
		if !opt.Available {
			warning = "This combination is reserved by another application or Windows."
		}

		hotkeyOptions = append(hotkeyOptions, Option{
			ID:               opt.Key,
			Label:            opt.Label,
			Badge:            badge,
			Available:        opt.Available,
			ShowAvailability: true,
			Warning:          warning,
		})

		if opt.Key == cfg.Hotkey {
			defaultHotkeyIdx = i
		}
	}

	hotkeyIdx, err := Select("Choose your global hotkey to activate Ganymede:", hotkeyOptions, defaultHotkeyIdx)
	if err != nil {
		return nil, err
	}
	cfg.Hotkey = hotkeyOptions[hotkeyIdx].ID

	// -------------------------------------------------------------
	// QUESTION 3: Activate on Startup
	// -------------------------------------------------------------
	startupOptions := []Option{
		{
			ID:          "yes",
			Label:       "Yes",
			Badge:       fmt.Sprintf("%s(Recommended)%s", banner.Color{R: 251, G: 191, B: 36}.RGB(), banner.Reset()),
			Description: "Ganymede listener runs silently on Windows login so hotkey always works",
		},
		{
			ID:          "no",
			Label:       "No",
			Description: "You must manually run 'ganymede start' in your terminal each session",
		},
	}

	defaultStartupIdx := 0
	if !cfg.Startup {
		defaultStartupIdx = 1
	}

	startupIdx, err := Select("Activate this CLI tool automatically on system startup?", startupOptions, defaultStartupIdx)
	if err != nil {
		return nil, err
	}
	cfg.Startup = (startupOptions[startupIdx].ID == "yes")

	// Apply startup registry setting
	if cfg.Startup {
		curExe, _ := os.Executable()
		_ = startup.Enable(curExe)
	} else {
		_ = startup.Disable()
	}

	// Save configuration to disk
	if err := config.Save(cfg); err != nil {
		return nil, fmt.Errorf("failed to save configuration: %w", err)
	}

	// Show illustration and welcome message AFTER configuration questions complete
	banner.PrintWelcomeBanner(cfg.Theme)

	// Summary Card
	PrintSetupSummary(cfg)

	return cfg, nil
}

// PrintSetupSummary displays a structured post-install summary card
func PrintSetupSummary(cfg *config.Config) {
	cyan := banner.Color{R: 56, G: 189, B: 248}.RGB()
	green := banner.Color{R: 52, G: 199, B: 89}.RGB()
	gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
	dim := banner.Color{R: 156, G: 163, B: 175}.RGB()

	cfgPath, _ := config.GetConfigPath()

	fmt.Println()
	fmt.Println("  " + dim + strings.Repeat("─", 68) + banner.Reset())
	fmt.Printf("  %s%s✨ Ganymede Setup Complete!%s\n", banner.Bold(), green, banner.Reset())
	fmt.Println("  " + dim + strings.Repeat("─", 68) + banner.Reset())
	fmt.Printf("  • %sColor Mode:%s       %s%s%s\n", banner.Bold(), banner.Reset(), cyan, strings.ToUpper(cfg.ColorMode), banner.Reset())
	fmt.Printf("  • %sEyedropper Hotkey:%s %s%s%s\n", banner.Bold(), banner.Reset(), gold, strings.ToUpper(cfg.Hotkey), banner.Reset())
	if cfg.Startup {
		fmt.Printf("  • %sWindows Startup:%s   %sEnabled (Background silent listener)%s\n", banner.Bold(), banner.Reset(), green, banner.Reset())
	} else {
		fmt.Printf("  • %sWindows Startup:%s   %sDisabled (Manual start required)%s\n", banner.Bold(), banner.Reset(), dim, banner.Reset())
	}
	fmt.Printf("  • %sConfig File:%s       %s%s%s\n", banner.Bold(), banner.Reset(), dim, cfgPath, banner.Reset())
	fmt.Println("  " + dim + strings.Repeat("─", 68) + banner.Reset())

	fmt.Println()
	fmt.Printf("  %sHandy Next Steps:%s\n", banner.Bold(), banner.Reset())
	fmt.Printf("  1. %sganymede pick%s     - Test color picking immediately on your screen\n", banner.Bold(), banner.Reset())
	fmt.Printf("  2. %sganymede start%s    - Start the background hotkey listener\n", banner.Bold(), banner.Reset())
	fmt.Printf("  3. %sganymede config%s   - Change color mode, hotkey, or startup preferences anytime\n", banner.Bold(), banner.Reset())
	fmt.Printf("  4. %sganymede help%s     - Explore full documentation of all commands & flags\n", banner.Bold(), banner.Reset())
	fmt.Println()
}

// PromptColorMode prompts for color mode selection
func PromptColorMode(currentMode string) (string, error) {
	colorOptions := []Option{
		{
			ID:    "hex",
			Label: "HEX",
		},
		{
			ID:    "rgb",
			Label: "RGB",
		},
		{
			ID:    "cmyk",
			Label: "CMYK",
		},
	}

	defaultColorIdx := 0
	for i, opt := range colorOptions {
		if opt.ID == currentMode {
			defaultColorIdx = i
			break
		}
	}

	colorIdx, err := Select("In which color mode do you want to copy the code to?", colorOptions, defaultColorIdx)
	if err != nil {
		return currentMode, err
	}
	return colorOptions[colorIdx].ID, nil
}

// PromptHotkey scans and prompts for hotkey selection
func PromptHotkey(currentHotkey string) (string, error) {
	fmt.Printf("  %s⠋ Scanning Windows hotkeys for system availability...%s",
		banner.Color{R: 147, G: 197, B: 253}.RGB(), banner.Reset())
	scannedPresets := hotkey.ScanPresetOptions()
	fmt.Print("\r\x1b[2K")

	var hotkeyOptions []Option
	defaultHotkeyIdx := 0

	for i, opt := range scannedPresets {
		var badge string
		var warning string

		if opt.Key == "win+alt+p" {
			gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
			badge = fmt.Sprintf("%s(Recommended)%s", gold, banner.Reset())
		}
		if !opt.Available {
			warning = "This combination is reserved by another application or Windows."
		}

		hotkeyOptions = append(hotkeyOptions, Option{
			ID:               opt.Key,
			Label:            opt.Label,
			Badge:            badge,
			Available:        opt.Available,
			ShowAvailability: true,
			Warning:          warning,
		})

		if opt.Key == currentHotkey {
			defaultHotkeyIdx = i
		}
	}

	hotkeyIdx, err := Select("Choose your global hotkey to activate Ganymede:", hotkeyOptions, defaultHotkeyIdx)
	if err != nil {
		return currentHotkey, err
	}
	return hotkeyOptions[hotkeyIdx].ID, nil
}

// PromptStartup prompts for startup setting
func PromptStartup(currentStartup bool) (bool, error) {
	startupOptions := []Option{
		{
			ID:          "yes",
			Label:       "Yes (Recommended)",
			Description: "Ganymede listener runs silently on Windows login so your hotkey is always ready",
		},
		{
			ID:          "no",
			Label:       "No",
			Description: "You must manually run 'ganymede start' in your terminal each time you want to use it",
		},
	}

	defaultStartupIdx := 0
	if !currentStartup {
		defaultStartupIdx = 1
	}

	startupIdx, err := Select("Activate this CLI tool automatically on system startup?", startupOptions, defaultStartupIdx)
	if err != nil {
		return currentStartup, err
	}
	return (startupOptions[startupIdx].ID == "yes"), nil
}

// PromptTheme prompts for banner color theme
func PromptTheme(currentTheme string) (string, error) {
	themes := []Option{
		{ID: "ganymede", Label: "Ganymede", Description: "Indigo -> Violet -> Hot Pink -> Amber Gold"},
		{ID: "cyberpunk", Label: "Cyberpunk", Description: "Neon Pink -> Royal Purple -> Electric Cyan"},
		{ID: "rainbow", Label: "Rainbow", Description: "Continuous spectrum from vivid Red to Deep Violet"},
		{ID: "sunset", Label: "Sunset", Description: "Deep Crimson -> Warm Gold"},
		{ID: "aurora", Label: "Aurora", Description: "Emerald Green -> Cyan -> Deep Violet"},
	}

	defaultIdx := 0
	for i, t := range themes {
		if t.ID == currentTheme {
			defaultIdx = i
			break
		}
	}

	idx, err := Select("Choose your terminal aesthetic & banner theme:", themes, defaultIdx)
	if err != nil {
		return currentTheme, err
	}
	return themes[idx].ID, nil
}

// RunConfigMenu provides an interactive menu to adjust any major input
func RunConfigMenu(cfg *config.Config) (*config.Config, error) {
	banner.PrintCompact()
	fmt.Printf("%sInteractive Settings Configuration%s\n\n", banner.Bold(), banner.Reset())

	menuOptions := []Option{
		{
			ID:          "mode",
			Label:       fmt.Sprintf("Color Mode (Current: %s)", strings.ToUpper(cfg.ColorMode)),
			Description: "Change active clipboard copy format (HEX, RGB, CMYK)",
		},
		{
			ID:          "hotkey",
			Label:       fmt.Sprintf("Eyedropper Hotkey (Current: %s)", strings.ToUpper(cfg.Hotkey)),
			Description: "Re-scan system and choose a free shortcut combination",
		},
		{
			ID:          "startup",
			Label:       fmt.Sprintf("Windows Startup (Current: %s)", map[bool]string{true: "Enabled", false: "Disabled"}[cfg.Startup]),
			Description: "Toggle automatic background listener on login",
		},
		{
			ID:          "theme",
			Label:       fmt.Sprintf("Color Theme (Current: %s)", cfg.Theme),
			Description: "Change CLI palette & ASCII banner theme",
		},
		{
			ID:          "sound",
			Label:       fmt.Sprintf("Sound Chime (Current: %s)", map[bool]string{true: "Enabled", false: "Disabled"}[cfg.Sound]),
			Description: "Toggle subtle audio feedback on color pick",
		},
		{
			ID:          "wizard",
			Label:       "Run Complete Setup Wizard",
			Description: "Step through all initial configuration questions again",
		},
	}

	idx, err := Select("What would you like to configure?", menuOptions, 0)
	if err != nil {
		return cfg, err
	}
	fmt.Println()

	switch menuOptions[idx].ID {
	case "mode":
		newMode, err := PromptColorMode(cfg.ColorMode)
		if err == nil {
			cfg.ColorMode = newMode
			_ = config.Save(cfg)
			fmt.Printf("  %s✔ Color mode updated to %s%s\n\n", banner.Bold(), strings.ToUpper(newMode), banner.Reset())
		}
	case "hotkey":
		newKey, err := PromptHotkey(cfg.Hotkey)
		if err == nil {
			cfg.Hotkey = newKey
			_ = config.Save(cfg)
			fmt.Printf("  %s✔ Hotkey updated to %s%s\n\n", banner.Bold(), strings.ToUpper(newKey), banner.Reset())
		}
	case "startup":
		newStart, err := PromptStartup(cfg.Startup)
		if err == nil {
			cfg.Startup = newStart
			if newStart {
				curExe, _ := os.Executable()
				_ = startup.Enable(curExe)
				fmt.Printf("  %s✔ Windows startup enabled!%s\n\n", banner.Bold(), banner.Reset())
			} else {
				_ = startup.Disable()
				fmt.Printf("  %s✔ Windows startup disabled.%s\n\n", banner.Bold(), banner.Reset())
			}
			_ = config.Save(cfg)
		}
	case "theme":
		newTheme, err := PromptTheme(cfg.Theme)
		if err == nil {
			cfg.Theme = newTheme
			_ = config.Save(cfg)
			banner.PrintBanner(newTheme, false)
			fmt.Printf("  %s✔ Palette theme updated to '%s'!%s\n\n", banner.Bold(), newTheme, banner.Reset())
		}
	case "sound":
		cfg.Sound = !cfg.Sound
		_ = config.Save(cfg)
		stateStr := "enabled"
		if !cfg.Sound {
			stateStr = "disabled"
		}
		fmt.Printf("  %s✔ Sound feedback %s!%s\n\n", banner.Bold(), stateStr, banner.Reset())
	case "wizard":
		return RunSetupWizard(cfg)
	}

	return cfg, nil
}
