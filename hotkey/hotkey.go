package hotkey

import (
	"fmt"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"
)

// Win32 modifier constants
const (
	MOD_ALT     = 0x0001
	MOD_CONTROL = 0x0002
	MOD_SHIFT   = 0x0004
	MOD_WIN     = 0x0008
	MOD_NOREPEAT = 0x4000

	WM_HOTKEY = 0x0312
	WM_QUIT   = 0x0012
)

type POINT struct {
	X, Y int32
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

var (
	user32         = syscall.NewLazyDLL("user32.dll")
	regHotKey      = user32.NewProc("RegisterHotKey")
	unregHotKey    = user32.NewProc("UnregisterHotKey")
	peekMessageW   = user32.NewProc("PeekMessageW")
	getMessageW    = user32.NewProc("GetMessageW")
	postQuitMessage = user32.NewProc("PostQuitMessage")
)

// HotkeyOption holds information and live availability for a hotkey choice
type HotkeyOption struct {
	Key         string // e.g. "win+alt+p"
	Label       string // e.g. "Win + Alt + P"
	Modifiers   uint32
	VK          uint32
	Description string
	Available   bool
}

// PresetOptions returns the 3 curated hotkey pairs
func PresetOptions() []HotkeyOption {
	return []HotkeyOption{
		{
			Key:         "win+alt+p",
			Label:       "Win + Alt + P",
			Modifiers:   MOD_WIN | MOD_ALT,
			VK:          0x50, // 'P'
			Description: "Recommended for designers: P for Palette / Pixel Pick",
		},
		{
			Key:         "ctrl+alt+c",
			Label:       "Ctrl + Alt + C",
			Modifiers:   MOD_CONTROL | MOD_ALT,
			VK:          0x43, // 'C'
			Description: "Classic shortcut: C for Color picker",
		},
		{
			Key:         "alt+shift+x",
			Label:       "Alt + Shift + X",
			Modifiers:   MOD_ALT | MOD_SHIFT,
			VK:          0x58, // 'X'
			Description: "Ergonomic alternative: X for eXtract",
		},
	}
}

// CheckAvailability tests if a hotkey combination can be registered on this system
func CheckAvailability(modifiers uint32, vk uint32) bool {
	testID := 9999
	// Try registering hotkey without MOD_NOREPEAT for test
	r, _, _ := regHotKey.Call(0, uintptr(testID), uintptr(modifiers), uintptr(vk))
	if r == 0 {
		return false
	}
	// Unregister immediately
	unregHotKey.Call(0, uintptr(testID))
	return true
}

// ScanPresetOptions scans all preset hotkeys and checks their live system availability
func ScanPresetOptions() []HotkeyOption {
	presets := PresetOptions()
	for i := range presets {
		presets[i].Available = CheckAvailability(presets[i].Modifiers, presets[i].VK)
	}
	return presets
}

// ParseHotkey parses a string like "win+alt+p" into modifiers and virtual key code
func ParseHotkey(keyStr string) (uint32, uint32, string, error) {
	keyStr = strings.ToLower(strings.TrimSpace(keyStr))
	parts := strings.Split(keyStr, "+")

	var mods uint32
	var vk uint32
	var labelParts []string

	for _, p := range parts {
		p = strings.TrimSpace(p)
		switch p {
		case "win", "windows", "super":
			mods |= MOD_WIN
			labelParts = append(labelParts, "Win")
		case "alt":
			mods |= MOD_ALT
			labelParts = append(labelParts, "Alt")
		case "ctrl", "control":
			mods |= MOD_CONTROL
			labelParts = append(labelParts, "Ctrl")
		case "shift":
			mods |= MOD_SHIFT
			labelParts = append(labelParts, "Shift")
		default:
			// Key code
			if len(p) == 1 {
				ch := rune(p[0])
				if ch >= 'a' && ch <= 'z' {
					vk = uint32(ch - 'a' + 'A')
					labelParts = append(labelParts, strings.ToUpper(p))
				} else if ch >= '0' && ch <= '9' {
					vk = uint32(ch)
					labelParts = append(labelParts, p)
				} else {
					return 0, 0, "", fmt.Errorf("unsupported key character '%s'", p)
				}
			} else {
				return 0, 0, "", fmt.Errorf("unknown key token '%s'", p)
			}
		}
	}

	if vk == 0 {
		return 0, 0, "", fmt.Errorf("hotkey string missing primary key letter (e.g. + P)")
	}

	return mods, vk, strings.Join(labelParts, " + "), nil
}

// Listen registers the global hotkey and enters message loop, calling onTrigger when pressed
func Listen(hotkeyKey string, onTrigger func(), stopChan <-chan struct{}) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	mods, vk, label, err := ParseHotkey(hotkeyKey)
	if err != nil {
		return fmt.Errorf("invalid hotkey format '%s': %w", hotkeyKey, err)
	}

	hotkeyID := 1
	// Register with MOD_NOREPEAT to prevent spamming
	r, _, _ := regHotKey.Call(0, uintptr(hotkeyID), uintptr(mods|MOD_NOREPEAT), uintptr(vk))
	if r == 0 {
		// Fallback without MOD_NOREPEAT
		r, _, _ = regHotKey.Call(0, uintptr(hotkeyID), uintptr(mods), uintptr(vk))
		if r == 0 {
			return fmt.Errorf("hotkey '%s' is already in use by another program or Windows. Run 'ganymede config hotkey' to pick an available one", label)
		}
	}
	defer unregHotKey.Call(0, uintptr(hotkeyID))

	const PM_REMOVE = 0x0001
	var msg MSG

	for {
		select {
		case <-stopChan:
			return nil
		default:
			// Process all pending messages for this thread
			for {
				res, _, _ := peekMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0, PM_REMOVE)
				if res == 0 {
					break
				}
				if msg.Message == WM_HOTKEY && msg.WParam == uintptr(hotkeyID) {
					// Hotkey triggered! Run concurrently so hotkey loop is never blocked
					go onTrigger()
				}
			}
			time.Sleep(20 * time.Millisecond)
		}
	}
}
