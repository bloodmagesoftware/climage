package downloads

import (
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"
)

func getDownloadsDir() (string, error) {
	// GUID for FOLDERID_Downloads: 374DE290-123F-4565-9164-39C4925E467B
	guid := [16]byte{
		0x90, 0xE2, 0x4D, 0x37, // Data1 (little-endian)
		0x3F, 0x12, // Data2
		0x65, 0x45, // Data3
		0x91, 0x64, 0x39, 0xC4, 0x92, 0x5E, 0x46, 0x7B, // Data4
	}

	modShell32 := syscall.NewLazyDLL("shell32.dll")
	procSHGetKnownFolderPath := modShell32.NewProc("SHGetKnownFolderPath")

	modOle32 := syscall.NewLazyDLL("ole32.dll")
	procCoTaskMemFree := modOle32.NewProc("CoTaskMemFree")

	var out uintptr
	hr, _, _ := procSHGetKnownFolderPath.Call(
		uintptr(unsafe.Pointer(&guid[0])),
		uintptr(0), // dwFlags
		uintptr(0), // hToken
		uintptr(unsafe.Pointer(&out)),
	)
	if hr == 0 && out != 0 {
		// Convert UTF-16 PWSTR to Go string
		path := syscall.UTF16PtrToString((*uint16)(unsafe.Pointer(out)))
		// free returned memory
		procCoTaskMemFree.Call(out)
		if path != "" {
			return path, nil
		}
	}

	// On error or unsupported environment, fall back to %USERPROFILE%\Downloads or $HOME/Downloads
	if up := os.Getenv("USERPROFILE"); up != "" {
		return filepath.Join(up, "Downloads"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", errors.New("could not determine Downloads folder")
	}
	return filepath.Join(home, "Downloads"), nil
}
