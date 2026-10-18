package picker

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"os"
	"path/filepath"
	"runtime"
	"sync/atomic"
	"syscall"
	"unsafe"

	"ganymede-cli/colorutil"
)

var (
	ErrCancelled = errors.New("color picking cancelled by user")
)

type POINT struct {
	X, Y int32
}

type RECT struct {
	Left, Top, Right, Bottom int32
}

type MSG struct {
	Hwnd    uintptr
	Message uint32
	WParam  uintptr
	LParam  uintptr
	Time    uint32
	Pt      POINT
}

type MSLLHOOKSTRUCT struct {
	Pt          POINT
	MouseData   uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type KBDLLHOOKSTRUCT struct {
	VkCode      uint32
	ScanCode    uint32
	Flags       uint32
	Time        uint32
	DwExtraInfo uintptr
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     uintptr
	HIcon         uintptr
	HCursor       uintptr
	HbrBackground uintptr
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       uintptr
}

type PAINTSTRUCT struct {
	Hdc         uintptr
	FErase      int32
	RcPaint     RECT
	FRestore    int32
	FIncUpdate  int32
	RgbReserved [32]byte
}

type BITMAPINFOHEADER struct {
	BiSize          uint32
	BiWidth         int32
	BiHeight        int32
	BiPlanes        uint16
	BiBitCount      uint16
	BiCompression   uint32
	BiSizeImage     uint32
	BiXPelsPerMeter int32
	BiYPelsPerMeter int32
	BiClrUsed       uint32
	BiClrImportant  uint32
}

type ICONINFO struct {
	FIcon    int32
	XHotspot uint32
	YHotspot uint32
	HbmMask  uintptr
	HbmColor uintptr
}

const (
	WS_POPUP          = 0x80000000
	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_LAYERED     = 0x00080000
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_NOACTIVATE  = 0x08000000

	LWA_COLORKEY      = 0x00000001
	COLORKEY_MAGENTA  = 0x00FF00FF

	WH_MOUSE_LL       = 14
	WH_KEYBOARD_LL    = 13

	WM_DESTROY        = 0x0002
	WM_PAINT          = 0x000F
	WM_USER           = 0x0400
	WM_USER_EXIT      = WM_USER + 1
	WM_USER_UPDATE    = WM_USER + 2

	WM_MOUSEMOVE      = 0x0200
	WM_LBUTTONDOWN    = 0x0201
	WM_RBUTTONDOWN    = 0x0204
	WM_KEYDOWN        = 0x0100
	WM_SYSKEYDOWN     = 0x0104

	VK_ESCAPE         = 0x1B
	VK_RETURN         = 0x0D
	VK_SPACE          = 0x20

	SM_XVIRTUALSCREEN  = 76
	SM_YVIRTUALSCREEN  = 77
	SM_CXVIRTUALSCREEN = 78
	SM_CYVIRTUALSCREEN = 79

	SWP_NOSIZE        = 0x0001
	SWP_NOACTIVATE    = 0x0010
	SWP_SHOWWINDOW    = 0x0040
	HWND_TOPMOST      = ^uintptr(0)

	SRCCOPY           = 0x00CC0020
	TRANSPARENT       = 1
	DT_LEFT           = 0x00000000
	DT_CENTER         = 0x00000001
	DT_VCENTER        = 0x00000004
	DT_SINGLELINE     = 0x00000020
	FW_NORMAL         = 400
	FW_SEMIBOLD       = 600
	FW_BOLD           = 700
	ANTIALIASED_QUALITY = 4
	DEFAULT_CHARSET   = 1

	GMEM_MOVEABLE     = 0x0002
	CF_UNICODETEXT    = 13

	OCR_NORMAL        = 32512
	OCR_IBEAM         = 32513
	OCR_WAIT          = 32514
	OCR_CROSS         = 32515
	OCR_SIZEALL       = 32646
	OCR_HAND          = 32649
	SPI_SETCURSORS    = 0x0057
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	gdi32                      = syscall.NewLazyDLL("gdi32.dll")
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")

	setProcessDpiAwarenessCtx  = user32.NewProc("SetProcessDpiAwarenessContext")
	registerClassExW           = user32.NewProc("RegisterClassExW")
	createWindowExW            = user32.NewProc("CreateWindowExW")
	defWindowProcW             = user32.NewProc("DefWindowProcW")
	destroyWindow              = user32.NewProc("DestroyWindow")
	setWindowPos               = user32.NewProc("SetWindowPos")
	setLayeredWindowAttributes = user32.NewProc("SetLayeredWindowAttributes")
	setWindowsHookExW          = user32.NewProc("SetWindowsHookExW")
	unhookWindowsHookEx        = user32.NewProc("UnhookWindowsHookEx")
	callNextHookEx             = user32.NewProc("CallNextHookEx")
	getMessageW                = user32.NewProc("GetMessageW")
	translateMessage           = user32.NewProc("TranslateMessage")
	dispatchMessageW           = user32.NewProc("DispatchMessageW")
	postMessageW               = user32.NewProc("PostMessageW")
	invalidateRect             = user32.NewProc("InvalidateRect")
	beginPaint                 = user32.NewProc("BeginPaint")
	endPaint                   = user32.NewProc("EndPaint")
	getDC                      = user32.NewProc("GetDC")
	releaseDC                  = user32.NewProc("ReleaseDC")
	createCompatibleDC         = gdi32.NewProc("CreateCompatibleDC")
	createDIBSection           = gdi32.NewProc("CreateDIBSection")
	selectObject               = gdi32.NewProc("SelectObject")
	bitBlt                     = gdi32.NewProc("BitBlt")
	deleteDC                   = gdi32.NewProc("DeleteDC")
	deleteObject               = gdi32.NewProc("DeleteObject")
	createSolidBrush           = gdi32.NewProc("CreateSolidBrush")
	createPen                  = gdi32.NewProc("CreatePen")
	createFontW                = gdi32.NewProc("CreateFontW")
	createBitmap               = gdi32.NewProc("CreateBitmap")
	createIconIndirect         = user32.NewProc("CreateIconIndirect")
	setSystemCursor            = user32.NewProc("SetSystemCursor")
	systemParamsInfo           = user32.NewProc("SystemParametersInfoW")
	copyIcon                   = user32.NewProc("CopyIcon")
	drawTextW                  = user32.NewProc("DrawTextW")
	setTextColor               = gdi32.NewProc("SetTextColor")
	setBkMode                  = gdi32.NewProc("SetBkMode")
	getCursorPos               = user32.NewProc("GetCursorPos")
	getSystemMetrics           = user32.NewProc("GetSystemMetrics")
	messageBeep                = user32.NewProc("MessageBeep")
	openClipboard              = user32.NewProc("OpenClipboard")
	closeClipboard             = user32.NewProc("CloseClipboard")
	emptyClipboard             = user32.NewProc("EmptyClipboard")
	setClipboardData           = user32.NewProc("SetClipboardData")
	globalAlloc                = kernel32.NewProc("GlobalAlloc")
	globalLock                 = kernel32.NewProc("GlobalLock")
	globalUnlock               = kernel32.NewProc("GlobalUnlock")
)

// Precomputed 32x32 Figma Eyedropper ARGB icon (hotspot at 4, 30)
// White halo outline, solid black bulb & stroke, white interior barrel
var figmaEyedropperPixels = [1024]uint32{
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFF000000, 0xFF000000, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFF000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0xFFFFFFFF, 0xFFFFFFFF, 0xFFFFFFFF, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
	0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000, 0x00000000,
}

// Active picker session state
var (
	isBusyPicking     atomic.Bool
	hMouseHook        uintptr
	hKbdHook          uintptr
	hwndLoupe         uintptr
	sessionActive     bool
	pickSuccess       bool
	pickedR, pickedG, pickedB uint8

	curX, curY        int32
	virtX, virtY, virtW, virtH int32

	// Fast 15x15 desktop sampler
	desktopDC         uintptr
	sampleDC          uintptr
	sampleBmp         uintptr
	sampleOldBmp      uintptr
	sampleBits        uintptr
	samplePixels      []uint32

	// Loupe double-buffer
	loupeDC           uintptr
	loupeBmp          uintptr
	loupeOldBmp       uintptr
	loupeBits         uintptr
	loupePixels       []uint32
	maxCardW          int32 = 380
	currentCardW      int32 = 280
	loupeH            int32 = 84

	// Cached GDI styling objects
	badgeFontBold     uintptr
	badgeFontReg      uintptr
	activeColorMode   string
	updatePending     atomic.Bool
	isClassRegistered bool
)

func initDPI() {
	if setProcessDpiAwarenessCtx.Find() == nil {
		setProcessDpiAwarenessCtx.Call(uintptr(^uint64(3))) // DPI_AWARENESS_CONTEXT_PER_MONITOR_AWARE_V2
	} else {
		setDpiAware := user32.NewProc("SetProcessDPIAware")
		setDpiAware.Call()
	}
}

func sampleAt(x, y int32) {
	bitBlt.Call(sampleDC, 0, 0, 15, 15, desktopDC, uintptr(x-7), uintptr(y-7), SRCCOPY)
}

// setupEyedropperCursor creates a hardware cursor matching Figma eyedropper
// and assigns it to system cursors during the picking session.
func setupEyedropperCursor() (func(), error) {
	curW, curH := 32, 32
	pixData := make([]uint32, curW*curH)
	copy(pixData, figmaEyedropperPixels[:])

	// Try loading from eyedropper-icon.png if present in root or exe dir
	possiblePaths := []string{
		"eyedropper-icon.png",
	}
	if exePath, err := os.Executable(); err == nil {
		possiblePaths = append(possiblePaths, filepath.Join(filepath.Dir(exePath), "eyedropper-icon.png"))
	}

	for _, p := range possiblePaths {
		if f, err := os.Open(p); err == nil {
			if srcImg, _, err := image.Decode(f); err == nil {
				scaled := image.NewRGBA(image.Rect(0, 0, curW, curH))
				srcB := srcImg.Bounds()
				sw, sh := srcB.Dx(), srcB.Dy()

				// Box filter scaling down to 32x32
				for y := 0; y < curH; y++ {
					for x := 0; x < curW; x++ {
						totalA := 0
						count := 0
						for dy := 0; dy < 3; dy++ {
							for dx := 0; dx < 3; dx++ {
								sx := (x*3 + dx) * sw / 96
								sy := (y*3 + dy) * sh / 96
								_, _, _, a := srcImg.At(sx, sy).RGBA()
								totalA += int(a >> 8)
								count++
							}
						}
						avgA := totalA / count
						if avgA > 25 {
							scaled.Set(x, y, color.RGBA{R: 0, G: 0, B: 0, A: uint8(avgA)})
						}
					}
				}

				// Find outside space
				outside := make([][]bool, curH)
				for i := range outside {
					outside[i] = make([]bool, curW)
				}
				queue := [][2]int{{0, 0}, {curW - 1, 0}, {0, curH - 1}, {curW - 1, curH - 1}}
				for _, pt := range queue {
					outside[pt[1]][pt[0]] = true
				}
				head := 0
				for head < len(queue) {
					curr := queue[head]
					head++
					cx, cy := curr[0], curr[1]
					for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
						nx, ny := cx+d[0], cy+d[1]
						if nx >= 0 && nx < curW && ny >= 0 && ny < curH {
							if !outside[ny][nx] && scaled.RGBAAt(nx, ny).A < 80 {
								outside[ny][nx] = true
								queue = append(queue, [2]int{nx, ny})
							}
						}
					}
				}

				// Generate Figma-styled ARGB buffer
				for y := 0; y < curH; y++ {
					for x := 0; x < curW; x++ {
						if outside[y][x] {
							hasStrokeNeighbor := false
							for dy := -1; dy <= 1; dy++ {
								for dx := -1; dx <= 1; dx++ {
									nx, ny := x+dx, y+dy
									if nx >= 0 && nx < curW && ny >= 0 && ny < curH {
										if scaled.RGBAAt(nx, ny).A >= 80 {
											hasStrokeNeighbor = true
											break
										}
									}
								}
								if hasStrokeNeighbor {
									break
								}
							}
							if hasStrokeNeighbor {
								pixData[y*curW+x] = 0xFFFFFFFF // White halo
							} else {
								pixData[y*curW+x] = 0x00000000
							}
						} else {
							if scaled.RGBAAt(x, y).A >= 80 {
								pixData[y*curW+x] = 0xFF000000 // Black stroke
							} else if x >= 18 && y <= 13 {
								pixData[y*curW+x] = 0xFF000000 // Black bulb interior
							} else {
								pixData[y*curW+x] = 0xFFFFFFFF // White barrel interior
							}
						}
					}
				}
			}
			f.Close()
			break
		}
	}

	dc, _, _ := getDC.Call(0)
	defer releaseDC.Call(0, dc)

	var bih BITMAPINFOHEADER
	bih.BiSize = uint32(unsafe.Sizeof(bih))
	bih.BiWidth = int32(curW)
	bih.BiHeight = -int32(curH) // top-down
	bih.BiPlanes = 1
	bih.BiBitCount = 32

	var bits uintptr
	hbmColor, _, _ := createDIBSection.Call(dc, uintptr(unsafe.Pointer(&bih)), 0, uintptr(unsafe.Pointer(&bits)), 0, 0)
	if hbmColor == 0 {
		return nil, fmt.Errorf("failed to create DIB section for cursor")
	}
	defer deleteObject.Call(hbmColor)

	copy(unsafe.Slice((*uint32)(unsafe.Pointer(bits)), curW*curH), pixData)

	hbmMask, _, _ := createBitmap.Call(uintptr(curW), uintptr(curH), 1, 1, 0)
	if hbmMask == 0 {
		return nil, fmt.Errorf("failed to create mask bitmap for cursor")
	}
	defer deleteObject.Call(hbmMask)

	var ii ICONINFO
	ii.FIcon = 0 // Cursor
	ii.XHotspot = 4
	ii.YHotspot = 30
	ii.HbmMask = hbmMask
	ii.HbmColor = hbmColor

	hCursor, _, _ := createIconIndirect.Call(uintptr(unsafe.Pointer(&ii)))
	if hCursor == 0 {
		return nil, fmt.Errorf("failed to create custom cursor")
	}

	targets := []uint32{OCR_NORMAL, OCR_IBEAM, OCR_HAND, OCR_CROSS, OCR_SIZEALL}
	for _, id := range targets {
		c, _, _ := copyIcon.Call(hCursor)
		if c != 0 {
			setSystemCursor.Call(c, uintptr(id))
		}
	}
	deleteObject.Call(hCursor)

	return func() {
		systemParamsInfo.Call(SPI_SETCURSORS, 0, 0, 0)
	}, nil
}

func mouseHookCallback(nCode int32, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 && sessionActive {
		ms := (*MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		switch wParam {
		case WM_MOUSEMOVE:
			curX = ms.Pt.X
			curY = ms.Pt.Y
			if !updatePending.Swap(true) && hwndLoupe != 0 {
				postMessageW.Call(hwndLoupe, WM_USER_UPDATE, 0, 0)
			}
			r, _, _ := callNextHookEx.Call(hMouseHook, uintptr(nCode), wParam, lParam)
			return r

		case WM_LBUTTONDOWN:
			curX = ms.Pt.X
			curY = ms.Pt.Y
			sampleAt(curX, curY)
			centerCol := samplePixels[7*15+7]
			pickedR = uint8((centerCol >> 16) & 0xFF)
			pickedG = uint8((centerCol >> 8) & 0xFF)
			pickedB = uint8(centerCol & 0xFF)
			pickSuccess = true
			sessionActive = false
			postMessageW.Call(hwndLoupe, WM_USER_EXIT, 0, 0)
			return 1 // Swallow mouse click

		case WM_RBUTTONDOWN:
			pickSuccess = false
			sessionActive = false
			postMessageW.Call(hwndLoupe, WM_USER_EXIT, 0, 0)
			return 1 // Swallow right click
		}
	}
	r, _, _ := callNextHookEx.Call(hMouseHook, uintptr(nCode), wParam, lParam)
	return r
}

func kbdHookCallback(nCode int32, wParam uintptr, lParam uintptr) uintptr {
	if nCode >= 0 && sessionActive && (wParam == WM_KEYDOWN || wParam == WM_SYSKEYDOWN) {
		kb := (*KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
		switch kb.VkCode {
		case VK_ESCAPE, 'Q', 'q':
			pickSuccess = false
			sessionActive = false
			postMessageW.Call(hwndLoupe, WM_USER_EXIT, 0, 0)
			return 1

		case VK_RETURN, VK_SPACE:
			sampleAt(curX, curY)
			centerCol := samplePixels[7*15+7]
			pickedR = uint8((centerCol >> 16) & 0xFF)
			pickedG = uint8((centerCol >> 8) & 0xFF)
			pickedB = uint8(centerCol & 0xFF)
			pickSuccess = true
			sessionActive = false
			postMessageW.Call(hwndLoupe, WM_USER_EXIT, 0, 0)
			return 1
		}
	}
	r, _, _ := callNextHookEx.Call(hKbdHook, uintptr(nCode), wParam, lParam)
	return r
}

func positionLoupe() {
	if hwndLoupe == 0 {
		return
	}
	// Figma card floats directly next to the cursor tip with zero gap
	lx := curX + 16
	ly := curY + 12

	// Flip if near right or bottom screen boundary
	if lx+currentCardW > virtX+virtW {
		lx = curX - currentCardW - 12
	}
	if ly+loupeH > virtY+virtH {
		ly = curY - loupeH - 12
	}
	if lx < virtX {
		lx = virtX + 4
	}
	if ly < virtY {
		ly = virtY + 4
	}

	setWindowPos.Call(hwndLoupe, HWND_TOPMOST, uintptr(lx), uintptr(ly), uintptr(currentCardW), uintptr(loupeH), SWP_SHOWWINDOW|SWP_NOACTIVATE)
}

func renderLoupeBuffer() {
	sampleAt(curX, curY)

	centerCol := samplePixels[7*15+7]
	r := uint8((centerCol >> 16) & 0xFF)
	g := uint8((centerCol >> 8) & 0xFF)
	b := uint8(centerCol & 0xFF)
	cObj := colorutil.RGB{R: r, G: g, B: b}
	formattedStr := cObj.Format(activeColorMode)

	// Dynamically calculate width based on text length
	if len(formattedStr) <= 9 {
		currentCardW = 280
	} else if len(formattedStr) <= 20 {
		currentCardW = 315
	} else {
		currentCardW = 360
	}

	// 1. Fill entire card buffer with transparent colorkey (Magenta = 0x00FF00FF)
	for i := range loupePixels {
		loupePixels[i] = COLORKEY_MAGENTA
	}

	setPix := func(x, y int32, c uint32) {
		if x >= 0 && x < currentCardW && y >= 0 && y < loupeH {
			loupePixels[y*maxCardW+x] = c
		}
	}

	// 2. Draw rounded rectangle card container (radius 16, height 84)
	cr := int32(16)
	cardBg := uint32(0x00FFFFFF)     // Pure white
	cardBorder := uint32(0x00CBD5E1) // Crisp slate border
	cardOuter := uint32(0x00E2E8F0)  // Soft anti-aliased edge

	for y := int32(0); y < loupeH; y++ {
		for x := int32(0); x < currentCardW; x++ {
			cx, cy := int32(-1), int32(-1)
			if x < cr && y < cr {
				cx, cy = cr, cr
			} else if x >= currentCardW-cr && y < cr {
				cx, cy = currentCardW-cr-1, cr
			} else if x < cr && y >= loupeH-cr {
				cx, cy = cr, loupeH-cr-1
			} else if x >= currentCardW-cr && y >= loupeH-cr {
				cx, cy = currentCardW-cr-1, loupeH-cr-1
			}

			if cx == -1 {
				if x == 0 || x == currentCardW-1 || y == 0 || y == loupeH-1 {
					setPix(x, y, cardBorder)
				} else {
					setPix(x, y, cardBg)
				}
			} else {
				dx := x - cx
				dy := y - cy
				d2 := dx*dx + dy*dy
				if d2 <= (cr-1)*(cr-1) {
					setPix(x, y, cardBg)
				} else if d2 <= cr*cr {
					setPix(x, y, cardBorder)
				} else if d2 <= (cr+1)*(cr+1) {
					setPix(x, y, cardOuter)
				}
			}
		}
	}

	// 3. Magnifier Loupe: 56x56 at (14, 14), radius 10, 11x11 grid of 5x5 cells
	lx0, ly0 := int32(14), int32(14)
	lSize := int32(56)
	lr := int32(10)
	gridLineCol := uint32(0x00E2E8F0)

	for gy := int32(-5); gy <= 5; gy++ {
		for gx := int32(-5); gx <= 5; gx++ {
			sampCol := samplePixels[(gy+7)*15+(gx+7)]
			if sampCol == COLORKEY_MAGENTA {
				sampCol = 0x00FE00FE
			}

			cellX := lx0 + (gx+5)*5
			cellY := ly0 + (gy+5)*5

			for py := cellY; py < cellY+5; py++ {
				for px := cellX; px < cellX+5; px++ {
					relX := px - lx0
					relY := py - ly0
					if relX < 0 || relX >= lSize || relY < 0 || relY >= lSize {
						continue
					}

					cx, cy := int32(-1), int32(-1)
					if relX < lr && relY < lr {
						cx, cy = lr, lr
					} else if relX >= lSize-lr && relY < lr {
						cx, cy = lSize-lr-1, lr
					} else if relX < lr && relY >= lSize-lr {
						cx, cy = lr, lSize-lr-1
					} else if relX >= lSize-lr && relY >= lSize-lr {
						cx, cy = lSize-lr-1, lSize-lr-1
					}

					if cx != -1 {
						dx := relX - cx
						dy := relY - cy
						if dx*dx+dy*dy > lr*lr {
							continue
						}
					}

					// Subtle gridline on right & bottom edge
					if px == cellX+4 || py == cellY+4 {
						setPix(px, py, gridLineCol)
					} else {
						setPix(px, py, sampCol)
					}
				}
			}
		}
	}

	// 4. Center Reticle framing active pixel (cell gx=0, gy=0)
	retX := lx0 + 5*5
	retY := ly0 + 5*5
	for x := retX - 1; x <= retX+5; x++ {
		setPix(x, retY-1, 0x00000000)
		setPix(x, retY+5, 0x00000000)
	}
	for y := retY - 1; y <= retY+5; y++ {
		setPix(retX-1, y, 0x00000000)
		setPix(retX+5, y, 0x00000000)
	}

	// 5. Rounded Color Swatch: 22x22 at (84, 18), radius 6
	swX0, swY0 := int32(84), int32(18)
	swSize := int32(22)
	swR := int32(6)
	swColor := uint32(b)<<16 | uint32(g)<<8 | uint32(r)
	if swColor == COLORKEY_MAGENTA {
		swColor = 0x00FE00FE
	}
	swBorder := uint32(0x00CBD5E1)

	for sy := int32(0); sy < swSize; sy++ {
		for sx := int32(0); sx < swSize; sx++ {
			px := swX0 + sx
			py := swY0 + sy

			cx, cy := int32(-1), int32(-1)
			if sx < swR && sy < swR {
				cx, cy = swR, swR
			} else if sx >= swSize-swR && sy < swR {
				cx, cy = swSize-swR-1, swR
			} else if sx < swR && sy >= swSize-swR {
				cx, cy = swR, swSize-swR-1
			} else if sx >= swSize-swR && sy >= swSize-swR {
				cx, cy = swSize-swR-1, swSize-swR-1
			}

			if cx == -1 {
				if sx == 0 || sx == swSize-1 || sy == 0 || sy == swSize-1 {
					setPix(px, py, swBorder)
				} else {
					setPix(px, py, swColor)
				}
			} else {
				dx := sx - cx
				dy := sy - cy
				d2 := dx*dx + dy*dy
				if d2 <= (swR-1)*(swR-1) {
					setPix(px, py, swColor)
				} else if d2 <= swR*swR {
					setPix(px, py, swBorder)
				}
			}
		}
	}

	// 6. Click Icon at (86, 50) with sparkle burst rays
	iconX, iconY := int32(86), int32(50)
	clickCol := uint32(0x0064748B) // Slate 500
	rays := [][2]int32{
		{iconX + 4, iconY},
		{iconX + 7, iconY + 1},
		{iconX + 9, iconY + 4},
		{iconX + 1, iconY + 1},
		{iconX, iconY + 4},
	}
	for _, pt := range rays {
		setPix(pt[0], pt[1], clickCol)
	}
	arrowPixs := [][2]int32{
		{iconX + 4, iconY + 4},
		{iconX + 5, iconY + 4},
		{iconX + 5, iconY + 5},
		{iconX + 6, iconY + 5},
		{iconX + 6, iconY + 6},
		{iconX + 7, iconY + 6},
		{iconX + 7, iconY + 7},
		{iconX + 8, iconY + 7},
		{iconX + 8, iconY + 8},
		{iconX + 6, iconY + 8},
		{iconX + 7, iconY + 9},
		{iconX + 8, iconY + 10},
		{iconX + 9, iconY + 11},
	}
	for _, pt := range arrowPixs {
		setPix(pt[0], pt[1], clickCol)
	}

	// 7. Typography via GDI (Bold Color code + Semibold "Click to sample")
	setBkMode.Call(loupeDC, TRANSPARENT)

	oldFont, _, _ := selectObject.Call(loupeDC, badgeFontBold)
	setTextColor.Call(loupeDC, 0x00141414) // Pure dark slate #141414 (no color fringing)
	hexText, _ := syscall.UTF16FromString(formattedStr)
	hexRect := RECT{Left: 116, Top: 16, Right: currentCardW - 10, Bottom: 42}
	drawTextW.Call(loupeDC, uintptr(unsafe.Pointer(&hexText[0])), uintptr(len(hexText)-1), uintptr(unsafe.Pointer(&hexRect)), DT_LEFT|DT_VCENTER|DT_SINGLELINE)

	selectObject.Call(loupeDC, badgeFontReg)
	setTextColor.Call(loupeDC, 0x00695547) // Slate 600 (#475569 in COLORREF)
	subText, _ := syscall.UTF16FromString("Click to sample")
	subRect := RECT{Left: 110, Top: 46, Right: currentCardW - 10, Bottom: 68}
	drawTextW.Call(loupeDC, uintptr(unsafe.Pointer(&subText[0])), uintptr(len(subText)-1), uintptr(unsafe.Pointer(&subRect)), DT_LEFT|DT_VCENTER|DT_SINGLELINE)

	selectObject.Call(loupeDC, oldFont)
}

func loupeWndProc(hwnd uintptr, msg uint32, wParam, lParam uintptr) uintptr {
	switch msg {
	case WM_USER_UPDATE:
		updatePending.Store(false)
		positionLoupe()
		renderLoupeBuffer()
		invalidateRect.Call(hwnd, 0, 0)
		return 0

	case WM_PAINT:
		var ps PAINTSTRUCT
		hdc, _, _ := beginPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		bitBlt.Call(hdc, 0, 0, uintptr(currentCardW), uintptr(loupeH), loupeDC, 0, 0, SRCCOPY)
		endPaint.Call(hwnd, uintptr(unsafe.Pointer(&ps)))
		return 0

	case WM_USER_EXIT:
		sessionActive = false
		destroyWindow.Call(hwnd)
		return 0

	case WM_DESTROY:
		hwndLoupe = 0
		sessionActive = false
		return 0
	}
	r, _, _ := defWindowProcW.Call(hwnd, uintptr(msg), wParam, lParam)
	return r
}

// SetClipboard copies text into Windows clipboard
func SetClipboard(text string) error {
	r, _, _ := openClipboard.Call(0)
	if r == 0 {
		return fmt.Errorf("failed to open clipboard")
	}
	defer closeClipboard.Call()

	emptyClipboard.Call()

	utf16Chars, err := syscall.UTF16FromString(text)
	if err != nil {
		return err
	}

	bytesLen := len(utf16Chars) * 2
	hMem, _, _ := globalAlloc.Call(GMEM_MOVEABLE, uintptr(bytesLen))
	if hMem == 0 {
		return fmt.Errorf("GlobalAlloc failed")
	}

	pData, _, _ := globalLock.Call(hMem)
	if pData == 0 {
		return fmt.Errorf("GlobalLock failed")
	}

	dest := unsafe.Slice((*uint16)(unsafe.Pointer(pData)), len(utf16Chars))
	copy(dest, utf16Chars)

	globalUnlock.Call(hMem)

	ret, _, _ := setClipboardData.Call(CF_UNICODETEXT, hMem)
	if ret == 0 {
		return fmt.Errorf("SetClipboardData failed")
	}
	return nil
}

// PlayBeep plays subtle audio feedback
func PlayBeep() {
	messageBeep.Call(0)
}

// Pick triggers the screen eyedropper and returns the sampled color and formatted string
func Pick(colorMode string) (*colorutil.RGB, string, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// Prevent re-entrant color picking sessions
	if !isBusyPicking.CompareAndSwap(false, true) {
		return nil, "", fmt.Errorf("eyedropper session already active")
	}
	defer isBusyPicking.Store(false)

	activeColorMode = colorMode
	pickSuccess = false

	initDPI()

	// Activate hardware eyedropper cursor
	cursorCleanup, err := setupEyedropperCursor()
	if err == nil && cursorCleanup != nil {
		defer cursorCleanup()
	}

	// 1. Get virtual multi-monitor desktop bounds
	vx, _, _ := getSystemMetrics.Call(SM_XVIRTUALSCREEN)
	vy, _, _ := getSystemMetrics.Call(SM_YVIRTUALSCREEN)
	vw, _, _ := getSystemMetrics.Call(SM_CXVIRTUALSCREEN)
	vh, _, _ := getSystemMetrics.Call(SM_CYVIRTUALSCREEN)

	virtX = int32(vx)
	virtY = int32(vy)
	virtW = int32(vw)
	virtH = int32(vh)

	desktopDC, _, _ = getDC.Call(0)
	defer releaseDC.Call(0, desktopDC)

	// 2. Micro-sampler DC for real-time 15x15 capture
	sampleDC, _, _ = createCompatibleDC.Call(desktopDC)
	defer deleteDC.Call(sampleDC)

	var sBih BITMAPINFOHEADER
	sBih.BiSize = uint32(unsafe.Sizeof(sBih))
	sBih.BiWidth = 15
	sBih.BiHeight = -15 // top-down
	sBih.BiPlanes = 1
	sBih.BiBitCount = 32
	sampleBmp, _, _ = createDIBSection.Call(desktopDC, uintptr(unsafe.Pointer(&sBih)), 0, uintptr(unsafe.Pointer(&sampleBits)), 0, 0)
	sampleOldBmp, _, _ = selectObject.Call(sampleDC, sampleBmp)
	defer func() {
		selectObject.Call(sampleDC, sampleOldBmp)
		deleteObject.Call(sampleBmp)
	}()
	samplePixels = unsafe.Slice((*uint32)(unsafe.Pointer(sampleBits)), 15*15)

	// 3. Double-buffered loupe memory DC (preallocated for max card width)
	loupeDC, _, _ = createCompatibleDC.Call(desktopDC)
	defer deleteDC.Call(loupeDC)

	var lBih BITMAPINFOHEADER
	lBih.BiSize = uint32(unsafe.Sizeof(lBih))
	lBih.BiWidth = maxCardW
	lBih.BiHeight = -loupeH // top-down
	lBih.BiPlanes = 1
	lBih.BiBitCount = 32
	loupeBmp, _, _ = createDIBSection.Call(desktopDC, uintptr(unsafe.Pointer(&lBih)), 0, uintptr(unsafe.Pointer(&loupeBits)), 0, 0)
	loupeOldBmp, _, _ = selectObject.Call(loupeDC, loupeBmp)
	defer func() {
		selectObject.Call(loupeDC, loupeOldBmp)
		deleteObject.Call(loupeBmp)
	}()
	loupePixels = unsafe.Slice((*uint32)(unsafe.Pointer(loupeBits)), int(maxCardW*loupeH))

	// Pre-create professional GDI fonts (reused with zero handle leak)
	hBold := int32(-22)
	fontName, _ := syscall.UTF16FromString("Segoe UI Variable Display")
	badgeFontBold, _, _ = createFontW.Call(
		uintptr(hBold), 0, 0, 0, FW_BOLD, 0, 0, 0,
		DEFAULT_CHARSET, 0, 0, ANTIALIASED_QUALITY, 0,
		uintptr(unsafe.Pointer(&fontName[0])),
	)
	defer deleteObject.Call(badgeFontBold)

	hSub := int32(-15)
	fontSubName, _ := syscall.UTF16FromString("Segoe UI Variable Text")
	badgeFontReg, _, _ = createFontW.Call(
		uintptr(hSub), 0, 0, 0, FW_SEMIBOLD, 0, 0, 0,
		DEFAULT_CHARSET, 0, 0, ANTIALIASED_QUALITY, 0,
		uintptr(unsafe.Pointer(&fontSubName[0])),
	)
	defer deleteObject.Call(badgeFontReg)

	// 4. Register loupe class once
	classNameU16, _ := syscall.UTF16FromString("GanymedeLoupeOverlay")
	if !isClassRegistered {
		var wc WNDCLASSEX
		wc.CbSize = uint32(unsafe.Sizeof(wc))
		wc.Style = 0x0003 // CS_HREDRAW | CS_VREDRAW
		wc.LpfnWndProc = syscall.NewCallback(loupeWndProc)
		wc.LpszClassName = &classNameU16[0]
		registerClassExW.Call(uintptr(unsafe.Pointer(&wc)))
		isClassRegistered = true
	}

	// 5. Create transparent layered floating loupe window
	titleU16, _ := syscall.UTF16FromString("Ganymede Eyedropper")
	hwnd, _, _ := createWindowExW.Call(
		WS_EX_TOPMOST|WS_EX_TOOLWINDOW|WS_EX_LAYERED|WS_EX_TRANSPARENT|WS_EX_NOACTIVATE,
		uintptr(unsafe.Pointer(&classNameU16[0])),
		uintptr(unsafe.Pointer(&titleU16[0])),
		WS_POPUP,
		0, 0, uintptr(currentCardW), uintptr(loupeH),
		0, 0, 0, 0,
	)
	if hwnd == 0 {
		return nil, "", fmt.Errorf("failed to create floating loupe window")
	}
	hwndLoupe = hwnd
	defer func() {
		if hwndLoupe != 0 {
			destroyWindow.Call(hwndLoupe)
			hwndLoupe = 0
		}
	}()

	// Enable colorkey transparency (Magenta)
	setLayeredWindowAttributes.Call(hwndLoupe, COLORKEY_MAGENTA, 0, LWA_COLORKEY)

	// 6. Install low-level mouse and keyboard hooks
	mouseCb := syscall.NewCallback(mouseHookCallback)
	hMouseHook, _, _ = setWindowsHookExW.Call(WH_MOUSE_LL, mouseCb, 0, 0)
	defer unhookWindowsHookEx.Call(hMouseHook)

	kbdCb := syscall.NewCallback(kbdHookCallback)
	hKbdHook, _, _ = setWindowsHookExW.Call(WH_KEYBOARD_LL, kbdCb, 0, 0)
	defer unhookWindowsHookEx.Call(hKbdHook)

	sessionActive = true
	updatePending.Store(false)

	// Initial render at current cursor
	var pt POINT
	getCursorPos.Call(uintptr(unsafe.Pointer(&pt)))
	curX = pt.X
	curY = pt.Y
	renderLoupeBuffer()
	positionLoupe()
	setWindowPos.Call(hwndLoupe, HWND_TOPMOST, 0, 0, uintptr(currentCardW), uintptr(loupeH), SWP_SHOWWINDOW|SWP_NOACTIVATE)

	// 7. Message pump for eyedropper session
	var msg MSG
	for sessionActive || hwndLoupe != 0 {
		r, _, _ := getMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		if int32(r) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&msg)))
		dispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
	}

	if !pickSuccess {
		return nil, "", ErrCancelled
	}

	picked := &colorutil.RGB{R: pickedR, G: pickedG, B: pickedB}
	formatted := picked.Format(colorMode)
	_ = SetClipboard(formatted)
	PlayBeep()

	return picked, formatted, nil
}
