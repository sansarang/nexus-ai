//go:build windows

package window

import (
	"syscall"
	"unsafe"
)

var (
	user32               = syscall.NewLazyDLL("user32.dll")
	getForegroundWindow  = user32.NewProc("GetForegroundWindow")
	setWindowPosProc     = user32.NewProc("SetWindowPos")
	getSystemMetricsProc = user32.NewProc("GetSystemMetrics")
	showWindowProc       = user32.NewProc("ShowWindow")
	keybd_event          = user32.NewProc("keybd_event")
)

func screenWidth() int32 {
	r, _, _ := getSystemMetricsProc.Call(0)
	return int32(r)
}

func screenHeight() int32 {
	r, _, _ := getSystemMetricsProc.Call(1)
	return int32(r)
}

func setWinPos(hwnd uintptr, x, y, w, h int32) {
	setWindowPosProc.Call(hwnd, 0,
		uintptr(x), uintptr(y), uintptr(w), uintptr(h),
		0x0040)
}

func ArrangeWindows(direction string) error {
	hwnd, _, _ := getForegroundWindow.Call()
	sw := screenWidth()
	sh := screenHeight()
	switch direction {
	case "left":
		setWinPos(hwnd, 0, 0, sw/2, sh)
	case "right":
		setWinPos(hwnd, sw/2, 0, sw/2, sh)
	case "top":
		setWinPos(hwnd, 0, 0, sw, sh/2)
	case "bottom":
		setWinPos(hwnd, 0, sh/2, sw, sh/2)
	case "maximize":
		showWindowProc.Call(hwnd, 3)
	case "center":
		w, h := sw*2/3, sh*2/3
		setWinPos(hwnd, (sw-w)/2, (sh-h)/2, w, h)
	case "minimize_all":
		// Win+D shortcut
		keybd_event.Call(0x5B, 0, 0, 0) // VK_LWIN down
		keybd_event.Call(0x44, 0, 0, 0) // D down
		keybd_event.Call(0x44, 0, 2, 0) // D up
		keybd_event.Call(0x5B, 0, 2, 0) // VK_LWIN up
	}
	_ = unsafe.Pointer(nil)
	return nil
}
