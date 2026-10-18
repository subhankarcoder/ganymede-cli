package banner

import (
	"fmt"
	"math"
	"os"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Color represents an RGB color
type Color struct {
	R, G, B uint8
}

// Preset themes
var (
	// Cosmic Ganymede: Deep space indigo -> Electric Azure -> Nebula Gold
	ThemeGanymede = []Color{
		{R: 99, G: 102, B: 241},  // Indigo
		{R: 139, G: 92, B: 246},  // Violet
		{R: 236, G: 72, B: 153},  // Hot Pink
		{R: 244, G: 114, B: 182}, // Rose
		{R: 251, G: 191, B: 36},  // Amber Gold
	}

	// Rainbow Spectrum: Vivid continuous spectrum
	ThemeRainbow = []Color{
		{R: 255, G: 59, B: 48},   // Red
		{R: 255, G: 149, B: 0},   // Orange
		{R: 255, G: 204, B: 0},   // Yellow
		{R: 52, G: 199, B: 89},   // Green
		{R: 0, G: 199, B: 190},   // Teal
		{R: 48, G: 176, B: 254},  // Light Blue
		{R: 88, G: 86, B: 214},   // Indigo
		{R: 175, G: 82, B: 222},  // Purple
		{R: 255, G: 45, B: 85},   // Pink
	}

	// Cyberpunk: Neon Pink -> Royal Purple -> Electric Cyan
	ThemeCyberpunk = []Color{
		{R: 255, G: 0, B: 127},
		{R: 186, G: 12, B: 248},
		{R: 0, G: 240, B: 255},
		{R: 0, G: 255, B: 170},
	}

	// Sunset: Deep Crimson -> Blood Orange -> Warm Gold
	ThemeSunset = []Color{
		{R: 255, G: 75, B: 43},
		{R: 255, G: 65, B: 108},
		{R: 255, G: 138, B: 0},
		{R: 249, G: 212, B: 35},
	}

	// Aurora: Emerald Green -> Cyan -> Deep Violet
	ThemeAurora = []Color{
		{R: 0, G: 242, B: 96},
		{R: 5, G: 117, B: 230},
		{R: 142, G: 45, B: 226},
	}
)

// ANSI block text for "GANYMEDE"
var logoLines = []string{
	`  ██████╗  █████╗ ███╗   ██╗██╗   ██╗███╗   ███╗███████╗██████╗ ███████╗`,
	` ██╔════╝ ██╔══██╗████╗  ██║╚██╗ ██╔╝████╗ ████║██╔════╝██╔══██╗██╔════╝`,
	` ██║  ███╗███████║██╔██╗ ██║ ╚████╔╝ ██╔████╔██║█████╗  ██║  ██║█████╗  `,
	` ██║   ██║██╔══██║██║╚██╗██║  ╚██╔╝  ██║╚██╔╝██║██╔══╝  ██║  ██║██╔══╝  `,
	` ╚██████╔╝██║  ██║██║ ╚████║   ██║   ██║ ╚═╝ ██║███████╗██████╔╝███████╗`,
	`  ╚═════╝ ╚═╝  ╚═╝╚═╝  ╚═══╝   ╚═╝   ╚═╝     ╚═╝╚══════╝╚═════╝ ╚══════╝`,
}

// Stylized Color Prism Emblem
var prismEmblem = []string{
	`                  ▲`,
	`                 /█\`,
	`                /███\`,
	`               /█████\`,
	`              /███████\`,
	`             ◄█████████►`,
	`              \███████/`,
	`               \█████/`,
	`                \███/`,
	`                 \█/`,
	`                  ▼`,
}

func init() {
	enableWindowsANSI()
}

// enableWindowsANSI ensures virtual terminal processing is enabled on Windows
func enableWindowsANSI() {
	if os.Getenv("TERM") == "dumb" {
		return
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	setConsoleMode := kernel32.NewProc("SetConsoleMode")
	getConsoleMode := kernel32.NewProc("GetConsoleMode")
	stdout := syscall.Stdout

	var mode uint32
	r1, _, _ := getConsoleMode.Call(uintptr(stdout), uintptr(unsafe.Pointer(&mode)))
	if r1 != 0 {
		// ENABLE_VIRTUAL_TERMINAL_PROCESSING = 0x0004
		const enableVT = 0x0004
		setConsoleMode.Call(uintptr(stdout), uintptr(mode|enableVT))
	}
}

// Interpolate computes an intermediate color along a multi-stop palette for t in [0, 1]
func Interpolate(palette []Color, t float64) Color {
	if len(palette) == 0 {
		return Color{255, 255, 255}
	}
	if len(palette) == 1 || t <= 0.0 {
		return palette[0]
	}
	if t >= 1.0 {
		return palette[len(palette)-1]
	}

	scaled := t * float64(len(palette)-1)
	idx := int(math.Floor(scaled))
	frac := scaled - float64(idx)

	if idx >= len(palette)-1 {
		return palette[len(palette)-1]
	}

	c1 := palette[idx]
	c2 := palette[idx+1]

	return Color{
		R: uint8(float64(c1.R) + frac*float64(int(c2.R)-int(c1.R))),
		G: uint8(float64(c1.G) + frac*float64(int(c2.G)-int(c1.G))),
		B: uint8(float64(c1.B) + frac*float64(int(c2.B)-int(c1.B))),
	}
}

// RGB returns the ANSI truecolor escape string for foreground
func (c Color) RGB() string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm", c.R, c.G, c.B)
}

// BgRGB returns the ANSI truecolor escape string for background
func (c Color) BgRGB() string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm", c.R, c.G, c.B)
}

// Reset returns the ANSI reset code
func Reset() string {
	return "\x1b[0m"
}

// Bold returns ANSI bold code
func Bold() string {
	return "\x1b[1m"
}

// Dim returns ANSI dim code
func Dim() string {
	return "\x1b[2m"
}

// GetTheme returns the palette for a given name
func GetTheme(name string) []Color {
	switch strings.ToLower(name) {
	case "rainbow", "prism":
		return ThemeRainbow
	case "cyberpunk", "neon":
		return ThemeCyberpunk
	case "sunset":
		return ThemeSunset
	case "aurora":
		return ThemeAurora
	default:
		return ThemeGanymede
	}
}

// Version represents the current release version of Ganymede CLI.
// Can be overridden at build time via -ldflags "-X ganymede-cli/banner.Version=x.y.z"
var Version = "0.1.0"

// PrintBanner prints the Ganymede CLI header
func PrintBanner(themeName string, animated bool) {
	palette := GetTheme(themeName)

	fmt.Println()
	vStr := Version
	if !strings.HasPrefix(vStr, "v") {
		vStr = "v" + vStr
	}
	badge := fmt.Sprintf(" %s%s✦ GANYMEDE ✦%s  %s%s%s",
		Bold(), Color{R: 251, G: 191, B: 36}.RGB(), Reset(),
		Dim(), vStr, Reset(),
	)

	divider := fmt.Sprintf("%s%s%s", Dim(), strings.Repeat("─", 74), Reset())
	fmt.Println("  " + divider)
	fmt.Println(" " + badge)
	fmt.Println("  " + divider)
	fmt.Println()

	// Animate lines if requested
	for y, line := range logoLines {
		runes := []rune(line)
		var sb strings.Builder
		sb.WriteString(" ")
		for x, r := range runes {
			if r == ' ' {
				sb.WriteRune(' ')
				continue
			}
			normX := float64(x) / float64(len(runes))
			normY := float64(y) / float64(len(logoLines))
			t := (normX + normY*0.35) / 1.35
			col := Interpolate(palette, t)
			sb.WriteString(col.RGB())
			sb.WriteRune(r)
		}
		sb.WriteString(Reset())
		fmt.Println(sb.String())

		if animated {
			time.Sleep(20 * time.Millisecond)
		}
	}
	fmt.Println()
}

// PrintWelcomeBanner prints the celebratory post-setup illustration and welcome message
func PrintWelcomeBanner(themeName string) {
	PrintBanner(themeName, true)
	fmt.Printf("   %s%s⚡ Welcome to Ganymede!%s  %sReady to sample, convert and export colors.%s\n\n",
		Bold(), Color{R: 52, G: 199, B: 89}.RGB(), Reset(),
		Dim(), Reset(),
	)
}

// PrintIllustration prints the illuminated color prism emblem with spectrum rays
func PrintIllustration(themeName string, animated bool) {
	palette := GetTheme(themeName)
	fmt.Println()
	for y, line := range prismEmblem {
		runes := []rune(line)
		var sb strings.Builder
		for x, r := range runes {
			if r == ' ' {
				sb.WriteRune(' ')
				continue
			}
			normX := float64(x) / float64(len(runes))
			normY := float64(y) / float64(len(prismEmblem))
			t := (normX + normY) / 2.0
			col := Interpolate(palette, t)
			sb.WriteString(col.RGB())
			sb.WriteRune(r)
		}
		sb.WriteString(Reset())
		fmt.Println(sb.String())

		if animated {
			time.Sleep(25 * time.Millisecond)
		}
	}
	fmt.Println()
}

// PrintCompact prints a clean one-line badge for subcommands
func PrintCompact() {
	fmt.Printf("%s%s🎨 ganymede%s %s»%s ",
		Bold(), Color{R: 139, G: 92, B: 246}.RGB(), Reset(),
		Dim(), Reset(),
	)
}
