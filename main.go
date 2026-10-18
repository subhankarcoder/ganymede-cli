package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"ganymede-cli/banner"
	"ganymede-cli/clihelp"
	"ganymede-cli/colorutil"
	"ganymede-cli/config"
	"ganymede-cli/hotkey"
	"ganymede-cli/picker"
	"ganymede-cli/prompt"
	"ganymede-cli/startup"
)

var (
	kernel32          = syscall.NewLazyDLL("kernel32.dll")
	user32            = syscall.NewLazyDLL("user32.dll")
	procAttachConsole = kernel32.NewProc("AttachConsole")
	procFreeConsole   = kernel32.NewProc("FreeConsole")
	getConsoleWindow  = kernel32.NewProc("GetConsoleWindow")
	showWindow        = user32.NewProc("ShowWindow")
)

const (
	attachParentProcess = ^uintptr(0) // (DWORD)-1 = 0xFFFFFFFF
	swHide              = 0
)

func init() {
	// Detect if launched in silent background mode (e.g. from Windows Startup)
	silent := false
	for _, arg := range os.Args[1:] {
		if arg == "--silent" || arg == "-silent" || arg == "/silent" {
			silent = true
			break
		}
	}

	if silent {
		// If started in silent background mode and a console window was attached/allocated,
		// hide and detach it immediately.
		hwnd, _, _ := getConsoleWindow.Call()
		if hwnd != 0 {
			showWindow.Call(hwnd, swHide)
		}
		procFreeConsole.Call()
	} else {
		// When compiled with -ldflags "-H=windowsgui", Windows does not attach a console
		// by default. Attach to the parent console (cmd / powershell / terminal) so that
		// CLI commands, menus, and banners render directly into the user's terminal.
		procAttachConsole.Call(attachParentProcess)
	}
}

func main() {
	themeFlag := flag.String("theme", "", "Color theme (ganymede, rainbow, cyberpunk, sunset, aurora)")
	animateFlag := flag.Bool("animate", true, "Animate the banner reveal")
	prismFlag := flag.Bool("prism", false, "Show the prism character illustration")
	helpFlag := flag.Bool("help", false, "Display help instructions")
	versionFlag := flag.Bool("version", false, "Display version information")
	flag.BoolVar(versionFlag, "v", false, "Display version information (shorthand)")
	silentFlag := flag.Bool("silent", false, "Silent execution mode for background startup")

	flag.Parse()
	args := flag.Args()

	if *versionFlag {
		fmt.Printf("Ganymede CLI v%s\n", strings.TrimPrefix(banner.Version, "v"))
		return
	}

	cfg, err := config.Load()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	activeTheme := cfg.Theme
	if *themeFlag != "" {
		activeTheme = *themeFlag
	}

	if *helpFlag {
		clihelp.PrintFullHelp(activeTheme)
		return
	}

	// If no arguments provided
	if len(args) == 0 {
		if *prismFlag {
			banner.PrintIllustration(activeTheme, *animateFlag)
		}
		banner.PrintBanner(activeTheme, *animateFlag)
		printInteractiveWelcome(cfg)
		return
	}

	cmd := strings.ToLower(args[0])
	subArgs := args[1:]

	switch cmd {
	case "version", "-v", "--version":
		fmt.Printf("Ganymede CLI v%s\n", strings.TrimPrefix(banner.Version, "v"))
		return

	case "init", "install", "setup":
		handleInit(cfg)

	case "pick", "capture", "eyedropper":
		handlePick(cfg)

	case "start", "listen", "daemon":
		silent := *silentFlag
		for _, a := range subArgs {
			if a == "--silent" || a == "-silent" || a == "/silent" {
				silent = true
				break
			}
		}
		handleStart(cfg, silent)

	case "config":
		handleConfig(subArgs, cfg)

	case "status":
		clihelp.PrintStatus(cfg)

	case "help":
		clihelp.PrintFullHelp(activeTheme)

	case "welcome":
		banner.PrintBanner(activeTheme, true)
		printInteractiveWelcome(cfg)

	case "prism":
		banner.PrintIllustration(activeTheme, true)
		banner.PrintBanner(activeTheme, false)

	case "demo":
		themes := []string{"ganymede", "cyberpunk", "rainbow", "sunset", "aurora"}
		for _, th := range themes {
			fmt.Printf("\n--- Theme: %s ---\n", strings.ToUpper(th))
			banner.PrintBanner(th, false)
		}

	case "hex":
		c := colorutil.RGB{R: 255, G: 87, B: 51}
		val := c.Hex()
		_ = picker.SetClipboard(val)
		banner.PrintCompact()
		fmt.Printf("Copied %s%s%s to clipboard! %s\n", banner.Bold(), val, banner.Reset(), c.ANSIColorBlock())

	case "rgb":
		c := colorutil.RGB{R: 99, G: 102, B: 241}
		val := c.RGBString()
		_ = picker.SetClipboard(val)
		banner.PrintCompact()
		fmt.Printf("Copied %s%s%s to clipboard! %s\n", banner.Bold(), val, banner.Reset(), c.ANSIColorBlock())

	case "cmyk":
		c := colorutil.RGB{R: 99, G: 102, B: 241}
		val := c.CMYKString()
		_ = picker.SetClipboard(val)
		banner.PrintCompact()
		fmt.Printf("Copied %s%s%s to clipboard! %s\n", banner.Bold(), val, banner.Reset(), c.ANSIColorBlock())

	default:
		banner.PrintCompact()
		fmt.Printf("Unknown command '%s'. Run 'ganymede help' for available commands.\n", cmd)
		os.Exit(1)
	}
}

func handleInit(cfg *config.Config) {
	_, err := prompt.RunSetupWizard(cfg)
	if err != nil {
		fmt.Printf("Setup encountered an error: %v\n", err)
	}
}

func handlePick(cfg *config.Config) {
	banner.PrintCompact()
	fmt.Printf("%sScreen eyedropper activated!%s %sClick anywhere to pick color (Esc to cancel)...%s\n",
		banner.Bold(), banner.Reset(),
		banner.Color{R: 156, G: 163, B: 175}.RGB(), banner.Reset(),
	)
	time.Sleep(80 * time.Millisecond)

	picked, formatted, err := picker.Pick(cfg.ColorMode)
	if err != nil {
		if err == picker.ErrCancelled {
			fmt.Printf("  %sEyedropper cancelled. No color copied.%s\n",
				banner.Color{R: 156, G: 163, B: 175}.RGB(), banner.Reset())
			return
		}
		fmt.Printf("  %sError picking color: %v%s\n",
			banner.Color{R: 239, G: 68, B: 68}.RGB(), err, banner.Reset())
		return
	}

	green := banner.Color{R: 52, G: 199, B: 89}.RGB()
	gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
	dim := banner.Color{R: 156, G: 163, B: 175}.RGB()

	fmt.Println()
	fmt.Printf("  %s%s✔ Copied %s to clipboard!%s\n", banner.Bold(), green, formatted, banner.Reset())
	fmt.Println("  " + dim + strings.Repeat("─", 50) + banner.Reset())
	fmt.Printf("  • Swatch:    %s\n", picked.ANSIColorBlock())
	fmt.Printf("  • HEX:       %s%s%s\n", banner.Bold(), picked.Hex(), banner.Reset())
	fmt.Printf("  • RGB:       %s\n", picked.RGBString())
	fmt.Printf("  • CMYK:      %s\n", picked.CMYKString())
	fmt.Printf("  • Active:    %s%s mode%s\n", gold, strings.ToUpper(cfg.ColorMode), banner.Reset())
	fmt.Println("  " + dim + strings.Repeat("─", 50) + banner.Reset())
	fmt.Println()
}

func handleStart(cfg *config.Config, silent bool) {
	if !silent {
		banner.PrintCompact()
		fmt.Printf("%sStarting background hotkey listener...%s\n", banner.Bold(), banner.Reset())

		mods, vk, label, err := hotkey.ParseHotkey(cfg.Hotkey)
		if err != nil {
			fmt.Printf("  %sError parsing hotkey '%s': %v%s\n",
				banner.Color{R: 239, G: 68, B: 68}.RGB(), cfg.Hotkey, err, banner.Reset())
			return
		}

		avail := hotkey.CheckAvailability(mods, vk)
		if !avail {
			fmt.Printf("  %s⚠️  Warning: Hotkey '%s' might be reserved or in conflict with another application.%s\n",
				banner.Color{R: 251, G: 191, B: 36}.RGB(), label, banner.Reset())
			fmt.Printf("     Run 'ganymede config hotkey' to pick an available alternative.\n\n")
		}

		green := banner.Color{R: 52, G: 199, B: 89}.RGB()
		cyan := banner.Color{R: 56, G: 189, B: 248}.RGB()
		gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
		dim := banner.Color{R: 156, G: 163, B: 175}.RGB()

		fmt.Println("  " + dim + strings.Repeat("─", 64) + banner.Reset())
		fmt.Printf("  %s%s✦ Ganymede Color Picker is Active in Background%s\n", banner.Bold(), green, banner.Reset())
		fmt.Println("  " + dim + strings.Repeat("─", 64) + banner.Reset())
		fmt.Printf("  • %sGlobal Shortcut:%s  %s%s%s\n", banner.Bold(), banner.Reset(), gold, label, banner.Reset())
		fmt.Printf("  • %sClipboard Mode:%s   %s%s%s\n", banner.Bold(), banner.Reset(), cyan, strings.ToUpper(cfg.ColorMode), banner.Reset())
		fmt.Printf("  • %sAction:%s           Press %s anywhere on screen to pick color\n", banner.Bold(), banner.Reset(), label)
		fmt.Printf("  • %sTo Exit:%s          Press %sCtrl + C%s in this window\n", banner.Bold(), banner.Reset(), banner.Bold(), banner.Reset())
		fmt.Println("  " + dim + strings.Repeat("─", 64) + banner.Reset())
		fmt.Println()
	}

	stopChan := make(chan struct{})
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		if !silent {
			fmt.Println("\nStopping Ganymede listener...")
		}
		close(stopChan)
	}()

	err := hotkey.Listen(cfg.Hotkey, func() {
		// Hotkey pressed!
		picked, formatted, err := picker.Pick(cfg.ColorMode)
		if err == nil && picked != nil {
			if !silent {
				now := time.Now().Format("15:04:05")
				green := banner.Color{R: 52, G: 199, B: 89}.RGB()
				dim := banner.Color{R: 156, G: 163, B: 175}.RGB()
				fmt.Printf("  [%s%s%s] %s%s✔ Copied %s to clipboard%s %s (%s | %s)\n",
					dim, now, banner.Reset(),
					banner.Bold(), green, formatted, banner.Reset(),
					picked.ANSIColorBlock(),
					picked.RGBString(),
					picked.CMYKString(),
				)
			}
		}
	}, stopChan)

	if err != nil {
		fmt.Printf("Listener error: %v\n", err)
	}
}

func handleConfig(args []string, cfg *config.Config) {
	if len(args) == 0 {
		_, err := prompt.RunConfigMenu(cfg)
		if err != nil {
			fmt.Printf("Configuration cancelled or error: %v\n", err)
		}
		return
	}

	sub := strings.ToLower(args[0])
	switch sub {
	case "mode", "format":
		if len(args) < 2 {
			newMode, err := prompt.PromptColorMode(cfg.ColorMode)
			if err == nil {
				cfg.ColorMode = newMode
				_ = config.Save(cfg)
				fmt.Printf("Color mode updated to %s\n", strings.ToUpper(newMode))
			}
			return
		}
		mode := strings.ToLower(args[1])
		if mode != "hex" && mode != "rgb" && mode != "cmyk" {
			fmt.Println("Invalid mode. Supported modes: hex, rgb, cmyk")
			return
		}
		cfg.ColorMode = mode
		if err := config.Save(cfg); err != nil {
			fmt.Printf("Error saving config: %v\n", err)
			return
		}
		banner.PrintCompact()
		fmt.Printf("Color copy mode updated to %s%s%s\n", banner.Bold(), strings.ToUpper(mode), banner.Reset())

	case "hotkey", "shortcut", "key":
		newKey, err := prompt.PromptHotkey(cfg.Hotkey)
		if err == nil {
			cfg.Hotkey = newKey
			_ = config.Save(cfg)
			banner.PrintCompact()
			fmt.Printf("Global hotkey updated to %s%s%s\n", banner.Bold(), strings.ToUpper(newKey), banner.Reset())
		}

	case "startup", "autostart":
		if len(args) < 2 {
			newStart, err := prompt.PromptStartup(cfg.Startup)
			if err == nil {
				cfg.Startup = newStart
				applyStartup(newStart, cfg)
			}
			return
		}
		val := strings.ToLower(args[1])
		if val == "enable" || val == "on" || val == "true" || val == "yes" {
			cfg.Startup = true
			applyStartup(true, cfg)
		} else if val == "disable" || val == "off" || val == "false" || val == "no" {
			cfg.Startup = false
			applyStartup(false, cfg)
		} else {
			fmt.Println("Usage: ganymede config startup <enable|disable>")
		}

	case "theme":
		if len(args) < 2 {
			newTheme, err := prompt.PromptTheme(cfg.Theme)
			if err == nil {
				cfg.Theme = newTheme
				_ = config.Save(cfg)
				banner.PrintBanner(newTheme, false)
				fmt.Printf("Theme updated to '%s'\n", newTheme)
			}
			return
		}
		themeName := strings.ToLower(args[1])
		cfg.Theme = themeName
		_ = config.Save(cfg)
		banner.PrintBanner(themeName, false)
		fmt.Printf("Theme set to '%s'\n", themeName)

	case "sound":
		if len(args) < 2 {
			cfg.Sound = !cfg.Sound
		} else {
			val := strings.ToLower(args[1])
			cfg.Sound = (val == "on" || val == "true" || val == "enable")
		}
		_ = config.Save(cfg)
		stateStr := "enabled"
		if !cfg.Sound {
			stateStr = "disabled"
		}
		banner.PrintCompact()
		fmt.Printf("Audio chime %s\n", stateStr)

	case "show", "list":
		clihelp.PrintStatus(cfg)

	default:
		fmt.Printf("Unknown config option '%s'. Supported: mode, hotkey, startup, theme, sound, show\n", sub)
	}
}

func applyStartup(enable bool, cfg *config.Config) {
	if enable {
		curExe, _ := os.Executable()
		if err := startup.Enable(curExe); err != nil {
			fmt.Printf("Error enabling startup registry: %v\n", err)
			return
		}
		_ = config.Save(cfg)
		banner.PrintCompact()
		fmt.Println("Windows startup enabled! Ganymede will run in background on login.")
	} else {
		if err := startup.Disable(); err != nil {
			fmt.Printf("Error disabling startup registry: %v\n", err)
			return
		}
		_ = config.Save(cfg)
		banner.PrintCompact()
		fmt.Println("Windows startup disabled.")
	}
}

func printInteractiveWelcome(cfg *config.Config) {
	cyan := banner.Color{R: 56, G: 189, B: 248}.RGB()
	gold := banner.Color{R: 251, G: 191, B: 36}.RGB()
	green := banner.Color{R: 52, G: 199, B: 89}.RGB()
	dim := banner.Color{R: 156, G: 163, B: 175}.RGB()

	_, _, label, _ := hotkey.ParseHotkey(cfg.Hotkey)

	fmt.Printf("   %sCurrent Configuration:%s\n", banner.Bold(), banner.Reset())
	fmt.Printf("   • Format: %s%s%s  • Hotkey: %s%s%s  • Startup: %s%s%s\n\n",
		cyan, strings.ToUpper(cfg.ColorMode), banner.Reset(),
		gold, label, banner.Reset(),
		green, map[bool]string{true: "Enabled", false: "Disabled"}[cfg.Startup], banner.Reset(),
	)

	fmt.Printf("   %sHandy Commands:%s\n", banner.Bold(), banner.Reset())
	fmt.Printf("   • %sganymede init%s       - Run interactive 3-step setup wizard\n", banner.Bold(), banner.Reset())
	fmt.Printf("   • %sganymede pick%s       - Immediately sample color anywhere on screen\n", banner.Bold(), banner.Reset())
	fmt.Printf("   • %sganymede start%s      - Start hotkey listener (%s)\n", banner.Bold(), banner.Reset(), label)
	fmt.Printf("   • %sganymede config%s     - Interactive menu to modify major inputs\n", banner.Bold(), banner.Reset())
	fmt.Printf("   • %sganymede status%s     - Inspect active settings and hotkey health\n", banner.Bold(), banner.Reset())
	fmt.Printf("   • %sganymede help%s       - Complete documentation of all commands\n", banner.Bold(), banner.Reset())
	fmt.Println()
	fmt.Printf("   %sTip: Run 'ganymede init' anytime to reconfigure color mode, hotkey, or startup!%s\n\n", dim, banner.Reset())
}