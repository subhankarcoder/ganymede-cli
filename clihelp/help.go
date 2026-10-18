package clihelp

import (
	"fmt"
	"strings"

	"ganymede-cli/banner"
	"ganymede-cli/config"
	"ganymede-cli/hotkey"
	"ganymede-cli/startup"
)

// PrintFullHelp prints the comprehensive user manual for Ganymede CLI
func PrintFullHelp(theme string) {
	banner.PrintBanner(theme, false)

	gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
	cyan := banner.Color{R: 56, G: 189, B: 248}.RGB()
	green := banner.Color{R: 52, G: 199, B: 89}.RGB()
	purple := banner.Color{R: 139, G: 92, B: 246}.RGB()
	dim := banner.Color{R: 156, G: 163, B: 175}.RGB()
	divider := dim + strings.Repeat("─", 74) + banner.Reset()

	fmt.Println("  " + divider)
	fmt.Printf("   %s%sGANYMEDE CLI - THE SYSTEM-WIDE SCREEN COLOR PICKER%s\n", banner.Bold(), gold, banner.Reset())
	fmt.Printf("   %sPrecision eyedropper for digital designers, UI engineers, and artists.%s\n", dim, banner.Reset())
	fmt.Println("  " + divider)
	fmt.Println()

	// Section 1: Quick Usage
	fmt.Printf("   %s%sUSAGE:%s\n", banner.Bold(), cyan, banner.Reset())
	fmt.Println("     ganymede [command] [subcommand / options]")
	fmt.Println()

	// Section 2: Core Commands
	fmt.Printf("   %s%sPRIMARY COMMANDS:%s\n", banner.Bold(), cyan, banner.Reset())
	printCmd("init, setup, install", "Launch the interactive Next.js-style setup wizard (3 steps)")
	printCmd("pick, eyedropper", "Trigger screen eyedropper immediately from CLI & copy color")
	printCmd("start, listen", "Start background listener for global hotkey (e.g. Win+Alt+P)")
	printCmd("status", "Show active color mode, hotkey status, and startup configuration")
	printCmd("config [subcommand]", "Interactively modify preferences or pass specific settings")
	printCmd("help, --help", "Display this comprehensive user guide and command manual")
	printCmd("demo", "Preview all available color themes (Ganymede, Rainbow, etc.)")
	fmt.Println()

	// Section 3: Configuration Subcommands
	fmt.Printf("   %s%sHANDY CONFIGURATION COMMANDS:%s\n", banner.Bold(), purple, banner.Reset())
	printCmd("config", "Open interactive menu to adjust any of your primary settings")
	printCmd("config mode <hex|rgb|cmyk>", "Switch active clipboard copy format")
	printCmd("config hotkey", "Interactive hotkey chooser with live system conflict checking")
	printCmd("config startup <enable|disable>", "Enable or disable automatic background launch on Windows startup")
	printCmd("config theme <name>", "Select palette theme (ganymede, rainbow, cyberpunk, sunset, aurora)")
	printCmd("config sound <on|off>", "Toggle audio chime feedback when sampling colors")
	printCmd("config show, list", "Print current settings JSON and configuration file path")
	fmt.Println()

	// Section 4: Hotkey Combinations
	fmt.Printf("   %s%sDEFAULT HOTKEY OPTIONS & LIVE CONFLICT CHECKING:%s\n", banner.Bold(), green, banner.Reset())
	fmt.Printf("     %sGanymede checks hotkey availability against Windows in real-time:%s\n", dim, banner.Reset())
	fmt.Println("     • Win + Alt + P    (Recommended) Hotkey for Palette & Precision Pick")
	fmt.Println("     • Ctrl + Alt + C   Classic standard shortcut for Color picker")
	fmt.Println("     • Alt + Shift + X  Ergonomic alternative for rapid eXtraction")
	fmt.Printf("     %s💡 If a hotkey is in use by another app, Ganymede flags it and guides you.%s\n", dim, banner.Reset())
	fmt.Println()

	// Section 5: Color Formats
	fmt.Printf("   %s%sSUPPORTED COLOR MODES:%s\n", banner.Bold(), gold, banner.Reset())
	fmt.Printf("     • %sHEX:%s  #6366F1                      (Modern web standard, CSS, Figma)\n", banner.Bold(), banner.Reset())
	fmt.Printf("     • %sRGB:%s  rgb(99, 102, 241)            (Canvas, WebGL, digital graphics)\n", banner.Bold(), banner.Reset())
	fmt.Printf("     • %sCMYK:%s cmyk(59%%, 58%%, 0%%, 5%%)     (Offset & digital print production)\n", banner.Bold(), banner.Reset())
	fmt.Println()

	// Section 6: Workflow & Tips
	fmt.Printf("   %s%sWORKFLOW & TIPS FOR DESIGNERS:%s\n", banner.Bold(), cyan, banner.Reset())
	fmt.Println("     1. While eyedropper is active, a real-time Loupe card follows the cursor.")
	fmt.Println("     2. The Loupe displays a 40x40 color swatch, live color code, and X/Y pixels.")
	fmt.Println("     3. Left-Click anywhere across any connected monitor to sample & copy.")
	fmt.Println("     4. Press ESC or Right-Click anytime to cancel without modifying clipboard.")
	fmt.Println("     5. Keep Ganymede running in the background for instant hotkey access.")
	fmt.Println()

	fmt.Println("  " + divider)
	cfgPath, _ := config.GetConfigPath()
	fmt.Printf("   %sConfiguration path:%s %s\n", dim, banner.Reset(), cfgPath)
	fmt.Println("  " + divider)
	fmt.Println()
}

func printCmd(name, description string) {
	fmt.Printf("     %s%-32s%s %s\n",
		banner.Bold(), name, banner.Reset(),
		description,
	)
}

// PrintStatus displays the current active runtime state of Ganymede
func PrintStatus(cfg *config.Config) {
	banner.PrintCompact()
	fmt.Printf("%sSystem Status%s\n", banner.Bold(), banner.Reset())

	green := banner.Color{R: 52, G: 199, B: 89}.RGB()
	red := banner.Color{R: 239, G: 68, B: 68}.RGB()
	cyan := banner.Color{R: 56, G: 189, B: 248}.RGB()
	gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
	dim := banner.Color{R: 156, G: 163, B: 175}.RGB()

	divider := dim + strings.Repeat("─", 60) + banner.Reset()
	fmt.Println("  " + divider)

	// Mode
	fmt.Printf("  • %sColor Copy Mode:%s      %s%s%s\n",
		banner.Bold(), banner.Reset(),
		cyan, strings.ToUpper(cfg.ColorMode), banner.Reset(),
	)

	// Hotkey & Live availability
	mods, vk, label, err := hotkey.ParseHotkey(cfg.Hotkey)
	hotkeyStatus := ""
	if err == nil {
		avail := hotkey.CheckAvailability(mods, vk)
		if avail {
			hotkeyStatus = fmt.Sprintf("%s (%s[✓ Available]%s)", label, green, banner.Reset())
		} else {
			hotkeyStatus = fmt.Sprintf("%s (%s[✗ In use or registered]%s)", label, red, banner.Reset())
		}
	} else {
		hotkeyStatus = cfg.Hotkey + " (Invalid format)"
	}
	fmt.Printf("  • %sEyedropper Hotkey:%s    %s%s%s\n",
		banner.Bold(), banner.Reset(),
		gold, hotkeyStatus, banner.Reset(),
	)

	// Startup
	startEnabled, startPath, _ := startup.IsEnabled()
	if startEnabled {
		fmt.Printf("  • %sWindows Startup:%s      %sEnabled%s (%s)\n",
			banner.Bold(), banner.Reset(),
			green, banner.Reset(),
			startPath,
		)
	} else {
		fmt.Printf("  • %sWindows Startup:%s      %sDisabled%s (run 'ganymede config startup enable')\n",
			banner.Bold(), banner.Reset(),
			dim, banner.Reset(),
		)
	}

	// Sound
	soundStr := fmt.Sprintf("%sEnabled%s", green, banner.Reset())
	if !cfg.Sound {
		soundStr = fmt.Sprintf("%sDisabled%s", dim, banner.Reset())
	}
	fmt.Printf("  • %sSound Feedback:%s       %s\n", banner.Bold(), banner.Reset(), soundStr)

	// Config file
	cfgPath, _ := config.GetConfigPath()
	fmt.Printf("  • %sConfig File:%s          %s%s%s\n",
		banner.Bold(), banner.Reset(),
		dim, cfgPath, banner.Reset(),
	)

	fmt.Println("  " + divider)
	fmt.Println()
}
