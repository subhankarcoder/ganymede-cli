package colorutil

import (
	"fmt"
	"math"
	"strings"
)

// RGB represents an 8-bit RGB color
type RGB struct {
	R, G, B uint8
}

// Hex returns the #RRGGBB representation
func (c RGB) Hex() string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// RGBString returns the rgb(r, g, b) representation
func (c RGB) RGBString() string {
	return fmt.Sprintf("rgb(%d, %d, %d)", c.R, c.G, c.B)
}

// CMYK computes Cyan, Magenta, Yellow, Key percentages (0-100)
func (c RGB) CMYK() (float64, float64, float64, float64) {
	rNorm := float64(c.R) / 255.0
	gNorm := float64(c.G) / 255.0
	bNorm := float64(c.B) / 255.0

	maxVal := math.Max(rNorm, math.Max(gNorm, bNorm))
	k := 1.0 - maxVal

	if k >= 1.0 {
		return 0, 0, 0, 100
	}

	cyan := (1.0 - rNorm - k) / (1.0 - k) * 100.0
	magenta := (1.0 - gNorm - k) / (1.0 - k) * 100.0
	yellow := (1.0 - bNorm - k) / (1.0 - k) * 100.0
	black := k * 100.0

	return math.Round(cyan), math.Round(magenta), math.Round(yellow), math.Round(black)
}

// CMYKString returns the cmyk(c%, m%, y%, k%) representation
func (c RGB) CMYKString() string {
	cVal, mVal, yVal, kVal := c.CMYK()
	return fmt.Sprintf("cmyk(%.0f%%, %.0f%%, %.0f%%, %.0f%%)", cVal, mVal, yVal, kVal)
}

// Format formats the color based on the specified mode (hex, rgb, cmyk)
func (c RGB) Format(mode string) string {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "rgb":
		return c.RGBString()
	case "cmyk":
		return c.CMYKString()
	case "hex":
		fallthrough
	default:
		return c.Hex()
	}
}

// ANSIColorBlock returns a truecolor colored block for terminal preview
func (c RGB) ANSIColorBlock() string {
	return fmt.Sprintf("\x1b[48;2;%d;%d;%dm   \x1b[0m", c.R, c.G, c.B)
}

// ANSITextColor returns an ANSI foreground truecolor string
func (c RGB) ANSITextColor(text string) string {
	return fmt.Sprintf("\x1b[38;2;%d;%d;%dm%s\x1b[0m", c.R, c.G, c.B, text)
}
