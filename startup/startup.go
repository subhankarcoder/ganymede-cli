package startup

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

const (
	AppName           = "GanymedeColorPicker"
	HKEY_CURRENT_USER = 0x80000001
	KEY_READ          = 0x20019
	KEY_WRITE         = 0x20006
	REG_SZ            = 1
)

var (
	advapi32        = syscall.NewLazyDLL("advapi32.dll")
	regOpenKeyExW   = advapi32.NewProc("RegOpenKeyExW")
	regQueryValueEx = advapi32.NewProc("RegQueryValueExW")
	regSetValueExW  = advapi32.NewProc("RegSetValueExW")
	regDeleteValueW = advapi32.NewProc("RegDeleteValueW")
	regCloseKey     = advapi32.NewProc("RegCloseKey")
)

// IsEnabled checks if Ganymede is registered in Windows Startup Run key
func IsEnabled() (bool, string, error) {
	subKey, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var hKey uintptr
	r, _, _ := regOpenKeyExW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(subKey)), 0, KEY_READ, uintptr(unsafe.Pointer(&hKey)))
	if r != 0 {
		return false, "", fmt.Errorf("failed to open Run registry key (code %d)", r)
	}
	defer regCloseKey.Call(hKey)

	valName, _ := syscall.UTF16PtrFromString(AppName)
	var valType uint32
	var bufSize uint32
	r, _, _ = regQueryValueEx.Call(hKey, uintptr(unsafe.Pointer(valName)), 0, uintptr(unsafe.Pointer(&valType)), 0, uintptr(unsafe.Pointer(&bufSize)))
	if r != 0 {
		return false, "", nil // Not in startup
	}

	buf := make([]uint16, bufSize/2)
	r, _, _ = regQueryValueEx.Call(hKey, uintptr(unsafe.Pointer(valName)), 0, uintptr(unsafe.Pointer(&valType)), uintptr(unsafe.Pointer(&buf[0])), uintptr(unsafe.Pointer(&bufSize)))
	if r != 0 {
		return false, "", nil
	}

	return true, syscall.UTF16ToString(buf), nil
}

// Enable registers Ganymede in Windows Startup Run key
func Enable(customExePath string) error {
	exePath := customExePath
	if exePath == "" {
		curExe, err := os.Executable()
		if err != nil {
			return fmt.Errorf("unable to determine executable path: %w", err)
		}
		exePath = curExe
	}

	// Ensure absolute path
	absPath, err := filepath.Abs(exePath)
	if err == nil {
		exePath = absPath
	}

	// Command to run on startup: "path\to\ganymede.exe" start --silent
	cmdValue := fmt.Sprintf("\"%s\" start --silent", exePath)

	subKey, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var hKey uintptr
	r, _, _ := regOpenKeyExW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(subKey)), 0, KEY_WRITE, uintptr(unsafe.Pointer(&hKey)))
	if r != 0 {
		return fmt.Errorf("failed to open Run registry key for writing (code %d)", r)
	}
	defer regCloseKey.Call(hKey)

	valName, _ := syscall.UTF16PtrFromString(AppName)
	valData, _ := syscall.UTF16FromString(cmdValue)
	dataBytes := len(valData) * 2

	r, _, _ = regSetValueExW.Call(
		hKey,
		uintptr(unsafe.Pointer(valName)),
		0,
		REG_SZ,
		uintptr(unsafe.Pointer(&valData[0])),
		uintptr(dataBytes),
	)
	if r != 0 {
		return fmt.Errorf("failed to write startup entry (code %d)", r)
	}

	return nil
}

// Disable removes Ganymede from Windows Startup Run key
func Disable() error {
	subKey, _ := syscall.UTF16PtrFromString(`Software\Microsoft\Windows\CurrentVersion\Run`)
	var hKey uintptr
	r, _, _ := regOpenKeyExW.Call(HKEY_CURRENT_USER, uintptr(unsafe.Pointer(subKey)), 0, KEY_WRITE, uintptr(unsafe.Pointer(&hKey)))
	if r != 0 {
		return fmt.Errorf("failed to open Run registry key (code %d)", r)
	}
	defer regCloseKey.Call(hKey)

	valName, _ := syscall.UTF16PtrFromString(AppName)
	r, _, _ = regDeleteValueW.Call(hKey, uintptr(unsafe.Pointer(valName)))
	// 2 = ERROR_FILE_NOT_FOUND, which means already disabled
	if r != 0 && r != 2 {
		return fmt.Errorf("failed to delete startup entry (code %d)", r)
	}

	return nil
}
