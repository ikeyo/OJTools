//go:build windows

package win32

import (
	"fmt"
	"syscall"
	"unsafe"
)

type HANDLE uintptr
type HINSTANCE HANDLE
type HMONITOR HANDLE
type HWND HANDLE
type HDC HANDLE
type HHOOK HANDLE
type HBRUSH HANDLE
type HCURSOR HANDLE
type HICON HANDLE
type HGDIOBJ HANDLE
type HBITMAP HANDLE
type HMENU HANDLE
type COLORREF uint32

type RECT struct {
	Left   int32
	Top    int32
	Right  int32
	Bottom int32
}

type POINT struct {
	X int32
	Y int32
}

type SIZE struct {
	CX int32
	CY int32
}

type MONITORINFOEX struct {
	CbSize    uint32
	RcMonitor RECT
	RcWork    RECT
	DwFlags   uint32
	SzDevice  [32]uint16
}

type WNDCLASSEX struct {
	CbSize        uint32
	Style         uint32
	LpfnWndProc   uintptr
	CbClsExtra    int32
	CbWndExtra    int32
	HInstance     HINSTANCE
	HIcon         HANDLE
	HCursor       HCURSOR
	HbrBackground HBRUSH
	LpszMenuName  *uint16
	LpszClassName *uint16
	HIconSm       HANDLE
}

type MSG struct {
	HWnd     HWND
	Message  uint32
	WParam   uintptr
	LParam   uintptr
	Time     uint32
	Pt       POINT
	LPrivate uint32
}

type PAINTSTRUCT struct {
	Hdc         HDC
	Erase       int32
	RcPaint     RECT
	Restore     int32
	IncUpdate   int32
	RgbReserved [32]byte
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

type WINDOWPLACEMENT struct {
	Length           uint32
	Flags            uint32
	ShowCmd          uint32
	PtMinPosition    POINT
	PtMaxPosition    POINT
	RcNormalPosition RECT
}

type NOTIFYICONDATA struct {
	CbSize            uint32
	HWnd              HWND
	UID               uint32
	UFlags            uint32
	UCallbackMessage  uint32
	HIcon             HICON
	SzTip             [128]uint16
	DwState           uint32
	DwStateMask       uint32
	SzInfo            [256]uint16
	UTimeoutOrVersion uint32
	SzInfoTitle       [64]uint16
	DwInfoFlags       uint32
	GuidItem          [16]byte
	HBalloonIcon      HICON
}

type BLENDFUNCTION struct {
	BlendOp             byte
	BlendFlags          byte
	SourceConstantAlpha byte
	AlphaFormat         byte
}

type BITMAPINFOHEADER struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

type RGBQUAD struct {
	Blue     byte
	Green    byte
	Red      byte
	Reserved byte
}

type BITMAPINFO struct {
	Header BITMAPINFOHEADER
	Colors [1]RGBQUAD
}

type GDIPlusStartupInput struct {
	GDIPlusVersion           uint32
	DebugEventCallback       uintptr
	SuppressBackgroundThread int32
	SuppressExternalCodecs   int32
}

const (
	MONITORINFOF_PRIMARY     = 0x00000001
	MONITOR_DEFAULTTONEAREST = 0x00000002

	MONITOR_DPI_TYPE_EFFECTIVE  = 0
	DWMWA_EXTENDED_FRAME_BOUNDS = 9

	CS_VREDRAW = 0x0001
	CS_HREDRAW = 0x0002

	WS_POPUP   = 0x80000000
	WS_VISIBLE = 0x10000000

	WS_EX_TOPMOST     = 0x00000008
	WS_EX_TRANSPARENT = 0x00000020
	WS_EX_TOOLWINDOW  = 0x00000080
	WS_EX_LAYERED     = 0x00080000
	WS_EX_NOACTIVATE  = 0x08000000

	SW_HIDE           = 0
	SW_SHOWNORMAL     = 1
	SW_SHOWMINIMIZED  = 2
	SW_SHOWMAXIMIZED  = 3
	SW_SHOWNOACTIVATE = 4
	SW_SHOW           = 5
	SW_MINIMIZE       = 6
	SW_RESTORE        = 9

	WM_CLOSE         = 0x0010
	WM_COMMAND       = 0x0111
	WM_DESTROY       = 0x0002
	WM_ERASEBKGND    = 0x0014
	WM_NCHITTEST     = 0x0084
	WM_PAINT         = 0x000F
	WM_KEYDOWN       = 0x0100
	WM_KEYUP         = 0x0101
	WM_SYSKEYDOWN    = 0x0104
	WM_SYSKEYUP      = 0x0105
	WM_APP           = 0x8000
	WM_MOUSEMOVE     = 0x0200
	WM_LBUTTONDOWN   = 0x0201
	WM_LBUTTONUP     = 0x0202
	WM_LBUTTONDBLCLK = 0x0203
	WM_RBUTTONUP     = 0x0205
	WM_TIMER         = 0x0113
	WM_MOUSEWHEEL    = 0x020A

	WH_KEYBOARD_LL = 13
	WH_MOUSE_LL    = 14
	HC_ACTION      = 0

	LLMHF_INJECTED = 0x00000001

	IDC_ARROW       = 32512
	IDI_APPLICATION = 32512

	VK_SHIFT     = 0x10
	VK_CONTROL   = 0x11
	VK_MENU      = 0x12
	VK_ESCAPE    = 0x1B
	VK_RETURN    = 0x0D
	VK_UP        = 0x26
	VK_DOWN      = 0x28
	VK_LEFT      = 0x25
	VK_RIGHT     = 0x27
	VK_ADD       = 0x6B
	VK_K         = 0x4B
	VK_LSHIFT    = 0xA0
	VK_RSHIFT    = 0xA1
	VK_LCONTROL  = 0xA2
	VK_RCONTROL  = 0xA3
	VK_LMENU     = 0xA4
	VK_RMENU     = 0xA5
	VK_SUBTRACT  = 0x6D
	VK_OEM_PLUS  = 0xBB
	VK_OEM_MINUS = 0xBD

	MK_SHIFT = 0x0004

	HTTRANSPARENT = ^uintptr(0)

	MF_STRING    = 0x00000000
	MF_GRAYED    = 0x00000001
	MF_DISABLED  = 0x00000002
	MF_CHECKED   = 0x00000008
	MF_POPUP     = 0x00000010
	MF_SEPARATOR = 0x00000800

	TPM_RIGHTBUTTON = 0x0002
	TPM_RETURNCMD   = 0x0100

	BKMODE_TRANSPARENT = 1
	PS_SOLID           = 0
	NULL_BRUSH         = 5
	BI_RGB             = 0
	DIB_RGB_COLORS     = 0

	LWA_COLORKEY = 0x00000001
	ULW_ALPHA    = 0x00000002
	AC_SRC_OVER  = 0x00
	AC_SRC_ALPHA = 0x01

	MB_OK = 0x00000000

	NIM_ADD    = 0x00000000
	NIM_MODIFY = 0x00000001
	NIM_DELETE = 0x00000002

	NIF_MESSAGE = 0x00000001
	NIF_ICON    = 0x00000002
	NIF_TIP     = 0x00000004

	PROCESS_QUERY_LIMITED_INFORMATION = 0x1000

	SWP_NOSIZE     = 0x0001
	SWP_NOZORDER   = 0x0004
	SWP_NOACTIVATE = 0x0010
	SWP_SHOWWINDOW = 0x0040
)

var (
	dpiAwarenessContextPerMonitorAwareV2 = ^uintptr(3)

	user32   = syscall.NewLazyDLL("user32.dll")
	gdi32    = syscall.NewLazyDLL("gdi32.dll")
	kernel32 = syscall.NewLazyDLL("kernel32.dll")
	shcore   = syscall.NewLazyDLL("shcore.dll")

	procBeginPaint                    = user32.NewProc("BeginPaint")
	procCallNextHookEx                = user32.NewProc("CallNextHookEx")
	procCreateWindowExW               = user32.NewProc("CreateWindowExW")
	procCreateIcon                    = user32.NewProc("CreateIcon")
	procDefWindowProcW                = user32.NewProc("DefWindowProcW")
	procDestroyWindow                 = user32.NewProc("DestroyWindow")
	procDestroyIcon                   = user32.NewProc("DestroyIcon")
	procDispatchMessageW              = user32.NewProc("DispatchMessageW")
	procDestroyMenu                   = user32.NewProc("DestroyMenu")
	procEndPaint                      = user32.NewProc("EndPaint")
	procEnumDisplayMonitors           = user32.NewProc("EnumDisplayMonitors")
	procEnumWindows                   = user32.NewProc("EnumWindows")
	procGetClassNameW                 = user32.NewProc("GetClassNameW")
	procGetClientRect                 = user32.NewProc("GetClientRect")
	procGetCursorPos                  = user32.NewProc("GetCursorPos")
	procGetForegroundWindow           = user32.NewProc("GetForegroundWindow")
	procGetMessageW                   = user32.NewProc("GetMessageW")
	procGetMonitorInfoW               = user32.NewProc("GetMonitorInfoW")
	procGetWindowRect                 = user32.NewProc("GetWindowRect")
	procGetWindowPlacement            = user32.NewProc("GetWindowPlacement")
	procGetWindowTextLengthW          = user32.NewProc("GetWindowTextLengthW")
	procGetWindowTextW                = user32.NewProc("GetWindowTextW")
	procGetWindowThreadProcessId      = user32.NewProc("GetWindowThreadProcessId")
	procInvalidateRect                = user32.NewProc("InvalidateRect")
	procIsWindowVisible               = user32.NewProc("IsWindowVisible")
	procIsWindowArranged              = user32.NewProc("IsWindowArranged")
	procAppendMenuW                   = user32.NewProc("AppendMenuW")
	procCreatePopupMenu               = user32.NewProc("CreatePopupMenu")
	procLoadIconW                     = user32.NewProc("LoadIconW")
	procLoadCursorW                   = user32.NewProc("LoadCursorW")
	procMessageBoxW                   = user32.NewProc("MessageBoxW")
	procPostQuitMessage               = user32.NewProc("PostQuitMessage")
	procRegisterClassExW              = user32.NewProc("RegisterClassExW")
	procReleaseCapture                = user32.NewProc("ReleaseCapture")
	procSetCapture                    = user32.NewProc("SetCapture")
	procSetCursorPos                  = user32.NewProc("SetCursorPos")
	procSetForegroundWindow           = user32.NewProc("SetForegroundWindow")
	procSetLayeredWindowAttributes    = user32.NewProc("SetLayeredWindowAttributes")
	procSetProcessDpiAwarenessContext = user32.NewProc("SetProcessDpiAwarenessContext")
	procSetTimer                      = user32.NewProc("SetTimer")
	procSetWindowPlacement            = user32.NewProc("SetWindowPlacement")
	procSetWindowPos                  = user32.NewProc("SetWindowPos")
	procSetWindowsHookExW             = user32.NewProc("SetWindowsHookExW")
	procShowWindow                    = user32.NewProc("ShowWindow")
	procTrackPopupMenu                = user32.NewProc("TrackPopupMenu")
	procTranslateMessage              = user32.NewProc("TranslateMessage")
	procUnhookWindowsHookEx           = user32.NewProc("UnhookWindowsHookEx")
	procUpdateWindow                  = user32.NewProc("UpdateWindow")
	procUpdateLayeredWindow           = user32.NewProc("UpdateLayeredWindow")
	procKillTimer                     = user32.NewProc("KillTimer")

	procCreateCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	procCreateDIBSection   = gdi32.NewProc("CreateDIBSection")
	procCreatePen          = gdi32.NewProc("CreatePen")
	procCreateSolidBrush   = gdi32.NewProc("CreateSolidBrush")
	procDeleteDC           = gdi32.NewProc("DeleteDC")
	procDeleteObject       = gdi32.NewProc("DeleteObject")
	procEllipse            = gdi32.NewProc("Ellipse")
	procFillRect           = user32.NewProc("FillRect")
	procGetStockObject     = gdi32.NewProc("GetStockObject")
	procLineTo             = gdi32.NewProc("LineTo")
	procMoveToEx           = gdi32.NewProc("MoveToEx")
	procPolygon            = gdi32.NewProc("Polygon")
	procSelectObject       = gdi32.NewProc("SelectObject")
	procSetBkMode          = gdi32.NewProc("SetBkMode")
	procSetTextColor       = gdi32.NewProc("SetTextColor")
	procTextOutW           = gdi32.NewProc("TextOutW")

	procGetModuleHandleW           = kernel32.NewProc("GetModuleHandleW")
	procGetConsoleWindow           = kernel32.NewProc("GetConsoleWindow")
	procOpenProcess                = kernel32.NewProc("OpenProcess")
	procCloseHandle                = kernel32.NewProc("CloseHandle")
	procQueryFullProcessImageNameW = kernel32.NewProc("QueryFullProcessImageNameW")
	procMonitorFromWindow          = user32.NewProc("MonitorFromWindow")

	procGetDpiForMonitor = shcore.NewProc("GetDpiForMonitor")

	shell32              = syscall.NewLazyDLL("shell32.dll")
	procShellNotifyIconW = shell32.NewProc("Shell_NotifyIconW")
	procExtractIconExW   = shell32.NewProc("ExtractIconExW")

	dwmapi                    = syscall.NewLazyDLL("dwmapi.dll")
	procDwmGetWindowAttribute = dwmapi.NewProc("DwmGetWindowAttribute")

	gdiplus                  = syscall.NewLazyDLL("gdiplus.dll")
	procGdiplusStartup       = gdiplus.NewProc("GdiplusStartup")
	procGdiplusShutdown      = gdiplus.NewProc("GdiplusShutdown")
	procGdipCreateFromHDC    = gdiplus.NewProc("GdipCreateFromHDC")
	procGdipDeleteGraphics   = gdiplus.NewProc("GdipDeleteGraphics")
	procGdipSetSmoothingMode = gdiplus.NewProc("GdipSetSmoothingMode")
	procGdipGraphicsClear    = gdiplus.NewProc("GdipGraphicsClear")
	procGdipCreateSolidFill  = gdiplus.NewProc("GdipCreateSolidFill")
	procGdipDeleteBrush      = gdiplus.NewProc("GdipDeleteBrush")
	procGdipFillPolygonI     = gdiplus.NewProc("GdipFillPolygonI")
	procGdipFillEllipseI     = gdiplus.NewProc("GdipFillEllipseI")
)

func (r RECT) Width() int32 {
	return r.Right - r.Left
}

func (r RECT) Height() int32 {
	return r.Bottom - r.Top
}

func RGB(r, g, b byte) COLORREF {
	return COLORREF(uint32(r) | (uint32(g) << 8) | (uint32(b) << 16))
}

func SetPerMonitorV2() error {
	r1, _, err := procSetProcessDpiAwarenessContext.Call(dpiAwarenessContextPerMonitorAwareV2)
	if r1 == 0 {
		return err
	}
	return nil
}

func GetModuleHandle() (HINSTANCE, error) {
	r1, _, err := procGetModuleHandleW.Call(0)
	if r1 == 0 {
		return 0, err
	}
	return HINSTANCE(r1), nil
}

func GetConsoleWindow() HWND {
	r1, _, _ := procGetConsoleWindow.Call()
	return HWND(r1)
}

func OpenProcess(desiredAccess uint32, inheritHandle bool, processID uint32) (HANDLE, error) {
	var inherit uintptr
	if inheritHandle {
		inherit = 1
	}

	r1, _, err := procOpenProcess.Call(uintptr(desiredAccess), inherit, uintptr(processID))
	if r1 == 0 {
		return 0, err
	}
	return HANDLE(r1), nil
}

func CloseHandle(handle HANDLE) {
	procCloseHandle.Call(uintptr(handle))
}

func QueryFullProcessImageName(process HANDLE) (string, error) {
	buf := make([]uint16, 32768)
	size := uint32(len(buf))
	r1, _, err := procQueryFullProcessImageNameW.Call(
		uintptr(process),
		0,
		uintptr(unsafe.Pointer(&buf[0])),
		uintptr(unsafe.Pointer(&size)),
	)
	if r1 == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf[:size]), nil
}

func EnumDisplayMonitors(callback func(HMONITOR, HDC, *RECT) bool) error {
	cb := syscall.NewCallback(func(hMonitor, hdc, lprcMonitor, _ uintptr) uintptr {
		var rect *RECT
		if lprcMonitor != 0 {
			rect = (*RECT)(unsafe.Pointer(lprcMonitor))
		}
		if callback(HMONITOR(hMonitor), HDC(hdc), rect) {
			return 1
		}
		return 0
	})

	r1, _, err := procEnumDisplayMonitors.Call(0, 0, cb, 0)
	if r1 == 0 {
		return err
	}
	return nil
}

func EnumWindows(callback func(HWND) bool) error {
	cb := syscall.NewCallback(func(hwnd, _ uintptr) uintptr {
		if callback(HWND(hwnd)) {
			return 1
		}
		return 0
	})

	r1, _, err := procEnumWindows.Call(cb, 0)
	if r1 == 0 {
		return err
	}
	return nil
}

func GetMonitorInfo(handle HMONITOR, info *MONITORINFOEX) error {
	info.CbSize = uint32(unsafe.Sizeof(*info))
	r1, _, err := procGetMonitorInfoW.Call(uintptr(handle), uintptr(unsafe.Pointer(info)))
	if r1 == 0 {
		return err
	}
	return nil
}

func IsWindowVisible(hwnd HWND) bool {
	r1, _, _ := procIsWindowVisible.Call(uintptr(hwnd))
	return r1 != 0
}

func IsWindowArranged(hwnd HWND) bool {
	if err := procIsWindowArranged.Find(); err != nil {
		return false
	}
	r1, _, _ := procIsWindowArranged.Call(uintptr(hwnd))
	return r1 != 0
}

func GetWindowText(hwnd HWND) (string, error) {
	length, _, _ := procGetWindowTextLengthW.Call(uintptr(hwnd))
	if length == 0 {
		return "", nil
	}

	buf := make([]uint16, int(length)+1)
	r1, _, err := procGetWindowTextW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r1 == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf[:r1]), nil
}

func GetClassName(hwnd HWND) (string, error) {
	buf := make([]uint16, 256)
	r1, _, err := procGetClassNameW.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&buf[0])), uintptr(len(buf)))
	if r1 == 0 {
		return "", err
	}
	return syscall.UTF16ToString(buf[:r1]), nil
}

func GetWindowPlacement(hwnd HWND, placement *WINDOWPLACEMENT) error {
	placement.Length = uint32(unsafe.Sizeof(*placement))
	r1, _, err := procGetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(placement)))
	if r1 == 0 {
		return err
	}
	return nil
}

func SetWindowPlacement(hwnd HWND, placement *WINDOWPLACEMENT) error {
	placement.Length = uint32(unsafe.Sizeof(*placement))
	r1, _, err := procSetWindowPlacement.Call(uintptr(hwnd), uintptr(unsafe.Pointer(placement)))
	if r1 == 0 {
		return err
	}
	return nil
}

func GetWindowThreadProcessId(hwnd HWND) uint32 {
	var pid uint32
	procGetWindowThreadProcessId.Call(uintptr(hwnd), uintptr(unsafe.Pointer(&pid)))
	return pid
}

func GetDpiForMonitor(handle HMONITOR) (uint32, uint32, error) {
	if err := procGetDpiForMonitor.Find(); err != nil {
		return 96, 96, nil
	}

	var dpiX uint32
	var dpiY uint32
	hr, _, _ := procGetDpiForMonitor.Call(
		uintptr(handle),
		uintptr(MONITOR_DPI_TYPE_EFFECTIVE),
		uintptr(unsafe.Pointer(&dpiX)),
		uintptr(unsafe.Pointer(&dpiY)),
	)
	if hr != 0 {
		return 0, 0, fmt.Errorf("GetDpiForMonitor failed: 0x%x", hr)
	}

	return dpiX, dpiY, nil
}

func LoadCursor(id uintptr) (HCURSOR, error) {
	r1, _, err := procLoadCursorW.Call(0, id)
	if r1 == 0 {
		return 0, err
	}
	return HCURSOR(r1), nil
}

func LoadIcon(id uintptr) (HICON, error) {
	r1, _, err := procLoadIconW.Call(0, id)
	if r1 == 0 {
		return 0, err
	}
	return HICON(r1), nil
}

func CreateIcon(width, height int32, planes, bitsPixel byte, andBits, xorBits []byte) (HICON, error) {
	var andPtr uintptr
	var xorPtr uintptr
	if len(andBits) > 0 {
		andPtr = uintptr(unsafe.Pointer(&andBits[0]))
	}
	if len(xorBits) > 0 {
		xorPtr = uintptr(unsafe.Pointer(&xorBits[0]))
	}

	r1, _, err := procCreateIcon.Call(
		0,
		uintptr(width),
		uintptr(height),
		uintptr(planes),
		uintptr(bitsPixel),
		andPtr,
		xorPtr,
	)
	if r1 == 0 {
		return 0, err
	}
	return HICON(r1), nil
}

func RegisterClassEx(class *WNDCLASSEX) (uint16, error) {
	r1, _, err := procRegisterClassExW.Call(uintptr(unsafe.Pointer(class)))
	if r1 == 0 {
		return 0, err
	}
	return uint16(r1), nil
}

func CreateWindowEx(exStyle uint32, className, title string, style uint32, x, y, width, height int32, parent HWND, menu HMENU, instance HINSTANCE, param uintptr) (HWND, error) {
	classNamePtr, err := syscall.UTF16PtrFromString(className)
	if err != nil {
		return 0, err
	}
	titlePtr, err := syscall.UTF16PtrFromString(title)
	if err != nil {
		return 0, err
	}

	r1, _, callErr := procCreateWindowExW.Call(
		uintptr(exStyle),
		uintptr(unsafe.Pointer(classNamePtr)),
		uintptr(unsafe.Pointer(titlePtr)),
		uintptr(style),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(parent),
		uintptr(menu),
		uintptr(instance),
		param,
	)
	if r1 == 0 {
		return 0, callErr
	}
	return HWND(r1), nil
}

func DefWindowProc(hwnd HWND, msg uint32, wParam, lParam uintptr) uintptr {
	r1, _, _ := procDefWindowProcW.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
	return r1
}

func ShowWindow(hwnd HWND, cmdShow int32) {
	procShowWindow.Call(uintptr(hwnd), uintptr(cmdShow))
}

func UpdateWindow(hwnd HWND) error {
	r1, _, err := procUpdateWindow.Call(uintptr(hwnd))
	if r1 == 0 {
		return err
	}
	return nil
}

func BeginPaint(hwnd HWND, paint *PAINTSTRUCT) (HDC, error) {
	r1, _, err := procBeginPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(paint)))
	if r1 == 0 {
		return 0, err
	}
	return HDC(r1), nil
}

func EndPaint(hwnd HWND, paint *PAINTSTRUCT) {
	procEndPaint.Call(uintptr(hwnd), uintptr(unsafe.Pointer(paint)))
}

func GetClientRect(hwnd HWND, rect *RECT) error {
	r1, _, err := procGetClientRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	if r1 == 0 {
		return err
	}
	return nil
}

func GetWindowRect(hwnd HWND, rect *RECT) error {
	r1, _, err := procGetWindowRect.Call(uintptr(hwnd), uintptr(unsafe.Pointer(rect)))
	if r1 == 0 {
		return err
	}
	return nil
}

func DwmGetExtendedFrameBounds(hwnd HWND, rect *RECT) error {
	if err := procDwmGetWindowAttribute.Find(); err != nil {
		return err
	}
	hr, _, _ := procDwmGetWindowAttribute.Call(
		uintptr(hwnd),
		uintptr(DWMWA_EXTENDED_FRAME_BOUNDS),
		uintptr(unsafe.Pointer(rect)),
		unsafe.Sizeof(*rect),
	)
	if hr != 0 {
		return fmt.Errorf("DwmGetWindowAttribute failed: 0x%x", hr)
	}
	return nil
}

func InvalidateRect(hwnd HWND, rect *RECT, erase bool) {
	var eraseInt uintptr
	if erase {
		eraseInt = 1
	}
	var rectPtr uintptr
	if rect != nil {
		rectPtr = uintptr(unsafe.Pointer(rect))
	}
	procInvalidateRect.Call(uintptr(hwnd), rectPtr, eraseInt)
}

func CreateSolidBrush(color COLORREF) (HBRUSH, error) {
	r1, _, err := procCreateSolidBrush.Call(uintptr(color))
	if r1 == 0 {
		return 0, err
	}
	return HBRUSH(r1), nil
}

func CreatePen(style uint32, width int32, color COLORREF) (HGDIOBJ, error) {
	r1, _, err := procCreatePen.Call(uintptr(style), uintptr(width), uintptr(color))
	if r1 == 0 {
		return 0, err
	}
	return HGDIOBJ(r1), nil
}

func CreateCompatibleDC(hdc HDC) (HDC, error) {
	r1, _, err := procCreateCompatibleDC.Call(uintptr(hdc))
	if r1 == 0 {
		return 0, err
	}
	return HDC(r1), nil
}

func DeleteDC(hdc HDC) {
	procDeleteDC.Call(uintptr(hdc))
}

func CreateDIBSection(width, height int32) (HBITMAP, unsafe.Pointer, error) {
	info := BITMAPINFO{
		Header: BITMAPINFOHEADER{
			Size:        uint32(unsafe.Sizeof(BITMAPINFOHEADER{})),
			Width:       width,
			Height:      -height,
			Planes:      1,
			BitCount:    32,
			Compression: BI_RGB,
		},
	}

	var bits unsafe.Pointer
	r1, _, err := procCreateDIBSection.Call(
		0,
		uintptr(unsafe.Pointer(&info)),
		uintptr(DIB_RGB_COLORS),
		uintptr(unsafe.Pointer(&bits)),
		0,
		0,
	)
	if r1 == 0 {
		return 0, nil, err
	}
	return HBITMAP(r1), bits, nil
}

func FillRect(hdc HDC, rect *RECT, brush HBRUSH) error {
	r1, _, err := procFillRect.Call(uintptr(hdc), uintptr(unsafe.Pointer(rect)), uintptr(brush))
	if r1 == 0 {
		return err
	}
	return nil
}

func DeleteObject(obj HGDIOBJ) {
	procDeleteObject.Call(uintptr(obj))
}

func SelectObject(hdc HDC, obj HGDIOBJ) HGDIOBJ {
	r1, _, _ := procSelectObject.Call(uintptr(hdc), uintptr(obj))
	return HGDIOBJ(r1)
}

func GetStockObject(index int32) HGDIOBJ {
	r1, _, _ := procGetStockObject.Call(uintptr(index))
	return HGDIOBJ(r1)
}

func SetTextColor(hdc HDC, color COLORREF) {
	procSetTextColor.Call(uintptr(hdc), uintptr(color))
}

func SetBkMode(hdc HDC, mode int32) {
	procSetBkMode.Call(uintptr(hdc), uintptr(mode))
}

func TextOut(hdc HDC, x, y int32, text string) error {
	utf16 := syscall.StringToUTF16(text)
	r1, _, err := procTextOutW.Call(
		uintptr(hdc),
		uintptr(x),
		uintptr(y),
		uintptr(unsafe.Pointer(&utf16[0])),
		uintptr(len(utf16)-1),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func MoveToEx(hdc HDC, x, y int32) {
	procMoveToEx.Call(uintptr(hdc), uintptr(x), uintptr(y), 0)
}

func LineTo(hdc HDC, x, y int32) {
	procLineTo.Call(uintptr(hdc), uintptr(x), uintptr(y))
}

func Ellipse(hdc HDC, left, top, right, bottom int32) {
	procEllipse.Call(uintptr(hdc), uintptr(left), uintptr(top), uintptr(right), uintptr(bottom))
}

func Polygon(hdc HDC, points []POINT) error {
	if len(points) == 0 {
		return nil
	}
	r1, _, err := procPolygon.Call(
		uintptr(hdc),
		uintptr(unsafe.Pointer(&points[0])),
		uintptr(len(points)),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func GDIPlusStartup() (uintptr, error) {
	input := GDIPlusStartupInput{GDIPlusVersion: 1}
	var token uintptr
	status, _, err := procGdiplusStartup.Call(
		uintptr(unsafe.Pointer(&token)),
		uintptr(unsafe.Pointer(&input)),
		0,
	)
	if status != 0 {
		return 0, err
	}
	return token, nil
}

func GDIPlusShutdown(token uintptr) {
	if token != 0 {
		procGdiplusShutdown.Call(token)
	}
}

func GDIPlusFillPolygon(hdc HDC, points []POINT, argb uint32) error {
	if len(points) < 3 {
		return nil
	}

	graphics, err := GDIPlusGraphicsFromHDC(hdc)
	if err != nil {
		return err
	}
	defer procGdipDeleteGraphics.Call(graphics)

	brush, err := GDIPlusSolidBrush(argb)
	if err != nil {
		return err
	}
	defer procGdipDeleteBrush.Call(brush)

	status, _, err := procGdipFillPolygonI.Call(
		graphics,
		brush,
		uintptr(unsafe.Pointer(&points[0])),
		uintptr(len(points)),
		0,
	)
	if status != 0 {
		return err
	}
	return nil
}

func GDIPlusFillEllipse(hdc HDC, x, y, width, height int32, argb uint32) error {
	graphics, err := GDIPlusGraphicsFromHDC(hdc)
	if err != nil {
		return err
	}
	defer procGdipDeleteGraphics.Call(graphics)

	brush, err := GDIPlusSolidBrush(argb)
	if err != nil {
		return err
	}
	defer procGdipDeleteBrush.Call(brush)

	status, _, err := procGdipFillEllipseI.Call(
		graphics,
		brush,
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
	)
	if status != 0 {
		return err
	}
	return nil
}

func GDIPlusGraphicsFromHDC(hdc HDC) (uintptr, error) {
	var graphics uintptr
	status, _, err := procGdipCreateFromHDC.Call(uintptr(hdc), uintptr(unsafe.Pointer(&graphics)))
	if status != 0 {
		return 0, err
	}
	// SmoothingModeAntiAlias = 4 in GDI+.
	procGdipSetSmoothingMode.Call(graphics, 4)
	return graphics, nil
}

func GDIPlusSolidBrush(argb uint32) (uintptr, error) {
	var brush uintptr
	status, _, err := procGdipCreateSolidFill.Call(uintptr(argb), uintptr(unsafe.Pointer(&brush)))
	if status != 0 {
		return 0, err
	}
	return brush, nil
}

func SetCapture(hwnd HWND) {
	procSetCapture.Call(uintptr(hwnd))
}

func ReleaseCapture() {
	procReleaseCapture.Call()
}

func DestroyWindow(hwnd HWND) {
	procDestroyWindow.Call(uintptr(hwnd))
}

func DestroyIcon(icon HICON) {
	procDestroyIcon.Call(uintptr(icon))
}

func CreatePopupMenu() (HMENU, error) {
	r1, _, err := procCreatePopupMenu.Call()
	if r1 == 0 {
		return 0, err
	}
	return HMENU(r1), nil
}

func AppendMenu(menu HMENU, flags uint32, itemID uintptr, text string) error {
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	r1, _, callErr := procAppendMenuW.Call(
		uintptr(menu),
		uintptr(flags),
		itemID,
		uintptr(unsafe.Pointer(textPtr)),
	)
	if r1 == 0 {
		return callErr
	}
	return nil
}

func DestroyMenu(menu HMENU) {
	procDestroyMenu.Call(uintptr(menu))
}

func GetCursorPos(point *POINT) error {
	r1, _, err := procGetCursorPos.Call(uintptr(unsafe.Pointer(point)))
	if r1 == 0 {
		return err
	}
	return nil
}

func GetForegroundWindow() HWND {
	r1, _, _ := procGetForegroundWindow.Call()
	return HWND(r1)
}

func MonitorFromWindow(hwnd HWND, flags uint32) HMONITOR {
	r1, _, _ := procMonitorFromWindow.Call(uintptr(hwnd), uintptr(flags))
	return HMONITOR(r1)
}

func SetForegroundWindow(hwnd HWND) {
	procSetForegroundWindow.Call(uintptr(hwnd))
}

func SetLayeredWindowAttributes(hwnd HWND, colorKey COLORREF, alpha byte, flags uint32) error {
	r1, _, err := procSetLayeredWindowAttributes.Call(uintptr(hwnd), uintptr(colorKey), uintptr(alpha), uintptr(flags))
	if r1 == 0 {
		return err
	}
	return nil
}

func UpdateLayeredWindowAlpha(hwnd HWND, x, y, width, height int32, src HDC) error {
	dstPos := POINT{X: x, Y: y}
	size := SIZE{CX: width, CY: height}
	srcPos := POINT{}
	blend := BLENDFUNCTION{
		BlendOp:             AC_SRC_OVER,
		SourceConstantAlpha: 255,
		AlphaFormat:         AC_SRC_ALPHA,
	}

	r1, _, err := procUpdateLayeredWindow.Call(
		uintptr(hwnd),
		0,
		uintptr(unsafe.Pointer(&dstPos)),
		uintptr(unsafe.Pointer(&size)),
		uintptr(src),
		uintptr(unsafe.Pointer(&srcPos)),
		0,
		uintptr(unsafe.Pointer(&blend)),
		uintptr(ULW_ALPHA),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func SetTimer(hwnd HWND, id uintptr, elapse uint32, callback uintptr) uintptr {
	r1, _, _ := procSetTimer.Call(uintptr(hwnd), id, uintptr(elapse), callback)
	return r1
}

func KillTimer(hwnd HWND, id uintptr) {
	procKillTimer.Call(uintptr(hwnd), id)
}

func SetWindowPos(hwnd, insertAfter HWND, x, y, width, height int32, flags uint32) error {
	r1, _, err := procSetWindowPos.Call(
		uintptr(hwnd),
		uintptr(insertAfter),
		uintptr(x),
		uintptr(y),
		uintptr(width),
		uintptr(height),
		uintptr(flags),
	)
	if r1 == 0 {
		return err
	}
	return nil
}

func TrackPopupMenu(menu HMENU, flags uint32, x, y int32, reserved int32, hwnd HWND, rect *RECT) uintptr {
	var rectPtr uintptr
	if rect != nil {
		rectPtr = uintptr(unsafe.Pointer(rect))
	}
	r1, _, _ := procTrackPopupMenu.Call(
		uintptr(menu),
		uintptr(flags),
		uintptr(x),
		uintptr(y),
		uintptr(reserved),
		uintptr(hwnd),
		rectPtr,
	)
	return r1
}

func MessageBox(hwnd HWND, text, caption string, flags uint32) error {
	textPtr, err := syscall.UTF16PtrFromString(text)
	if err != nil {
		return err
	}
	captionPtr, err := syscall.UTF16PtrFromString(caption)
	if err != nil {
		return err
	}
	_, _, callErr := procMessageBoxW.Call(
		uintptr(hwnd),
		uintptr(unsafe.Pointer(textPtr)),
		uintptr(unsafe.Pointer(captionPtr)),
		uintptr(flags),
	)
	if callErr != syscall.Errno(0) {
		return callErr
	}
	return nil
}

func PostQuitMessage(code int32) {
	procPostQuitMessage.Call(uintptr(code))
}

func RunMessageLoop() error {
	var msg MSG
	for {
		r1, _, err := procGetMessageW.Call(uintptr(unsafe.Pointer(&msg)), 0, 0, 0)
		switch int32(r1) {
		case -1:
			return err
		case 0:
			return nil
		default:
			procTranslateMessage.Call(uintptr(unsafe.Pointer(&msg)))
			procDispatchMessageW.Call(uintptr(unsafe.Pointer(&msg)))
		}
	}
}

func SetWindowsHookEx(idHook int32, callback uintptr, instance HINSTANCE, threadID uint32) (HHOOK, error) {
	r1, _, err := procSetWindowsHookExW.Call(
		uintptr(idHook),
		callback,
		uintptr(instance),
		uintptr(threadID),
	)
	if r1 == 0 {
		return 0, err
	}
	return HHOOK(r1), nil
}

func CallNextHookEx(hhk HHOOK, nCode int32, wParam, lParam uintptr) uintptr {
	r1, _, _ := procCallNextHookEx.Call(uintptr(hhk), uintptr(nCode), wParam, lParam)
	return r1
}

func UnhookWindowsHookEx(hhk HHOOK) {
	procUnhookWindowsHookEx.Call(uintptr(hhk))
}

func SetCursorPos(x, y int32) error {
	r1, _, err := procSetCursorPos.Call(uintptr(x), uintptr(y))
	if r1 == 0 {
		return err
	}
	return nil
}

func ShellNotifyIcon(message uint32, data *NOTIFYICONDATA) error {
	r1, _, err := procShellNotifyIconW.Call(uintptr(message), uintptr(unsafe.Pointer(data)))
	if r1 == 0 {
		return err
	}
	return nil
}

func ExtractSmallIcon(path string) (HICON, error) {
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}

	var icon HICON
	r1, _, callErr := procExtractIconExW.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		0,
		0,
		uintptr(unsafe.Pointer(&icon)),
		1,
	)
	if r1 == 0 || icon == 0 {
		return 0, callErr
	}
	return icon, nil
}

func SetNotifyIconTip(data *NOTIFYICONDATA, tip string) {
	utf16 := syscall.StringToUTF16(tip)
	limit := len(data.SzTip)
	if len(utf16) < limit {
		limit = len(utf16)
	}
	for i := 0; i < limit; i++ {
		data.SzTip[i] = utf16[i]
	}
	if limit == len(data.SzTip) {
		data.SzTip[len(data.SzTip)-1] = 0
	}
}

func HiWord(value uintptr) uint16 {
	return uint16((value >> 16) & 0xffff)
}

func LoWord(value uintptr) uint16 {
	return uint16(value & 0xffff)
}

func SignedHiWord(value uintptr) int16 {
	return int16(HiWord(value))
}
