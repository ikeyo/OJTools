//go:build windows

package remap

import (
	"fmt"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"ojtools/internal/config"
	"ojtools/internal/highlight"
	"ojtools/internal/monitor"
	"ojtools/internal/win32"
)

var (
	activeService *Service
	hookProc      = syscall.NewCallback(lowLevelMouseProc)
	keyboardProc  = syscall.NewCallback(lowLevelKeyboardProc)
	trayProc      = syscall.NewCallback(trayWindowProc)
)

const (
	trayClassName                = "OJToolsTrayWindow"
	trayCallbackMessage          = win32.WM_APP + 1
	trayIconID                   = 1
	trayCalibrationCommand       = 997
	trayResetCalibrationCommand  = 998
	trayPauseCommand             = 999
	trayAutoStartCommand         = 1000
	trayRestoreOnLaunchCommand   = 1001
	trayShakeHighlightCommand    = 1002
	trayShakeDelay500Command     = 1003
	trayShakeDelay1000Command    = 1004
	trayShakeDelay2000Command    = 1005
	traySaveLayoutCommand        = 1006
	trayRestoreLayoutCommand     = 1007
	trayClearLayoutCommand       = 1008
	trayOpenConfigCommand        = 1009
	trayAboutCommand             = 1010
	trayExitCommand              = 1011
	trayLockKeyboardCommand      = 1012
	trayBlockedKeyStatusCommand  = 1013
	trayToggleBlockedKeyCommand  = 1014
	trayCaptureBlockedKeyCommand = 1015
	trayClearBlockedKeyCommand   = 1016

	shakeWindow        = 750 * time.Millisecond
	shakeMinSegment    = 10.0
	shakeMinPath       = 240.0
	shakeMaxNetRatio   = 0.45
	shakeMinTurns      = 4
	shakeSuppressTTL   = 150 * time.Millisecond
	fullscreenSlack    = int32(2)
	keyboardLockHotkey = "Ctrl+Alt+Shift+K"
)

type shakeSample struct {
	Point win32.POINT
	At    time.Time
}

type Service struct {
	Pair                  monitor.Pair
	Transform             config.Transform
	ConfigPath            string
	Config                config.Config
	Hook                  win32.HHOOK
	KeyboardHook          win32.HHOOK
	HWND                  win32.HWND
	Tray                  win32.NOTIFYICONDATA
	Icon                  win32.HICON
	Prev                  win32.POINT
	HavePrev              bool
	Paused                bool
	KeyboardLocked        bool
	CaptureNextBlockedKey bool
	CtrlDown              bool
	AltDown               bool
	ShiftDown             bool
	Highlight             *highlight.Manager
	ShakeSamples          []shakeSample
	ShakeSuppressUntil    time.Time
	ShakeSuppressed       bool
}

func Run(pair monitor.Pair, transform config.Transform, cfgPath string, cfg config.Config) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := win32.SetPerMonitorV2(); err != nil {
		// The process may already be DPI aware.
	}

	instance, err := win32.GetModuleHandle()
	if err != nil {
		return err
	}

	service := &Service{
		Pair:       pair,
		Transform:  transform,
		ConfigPath: cfgPath,
		Config:     cfg,
	}
	service.Config.Features.ShakeHighlightDelayMs = normalizeShakeDelay(service.Config.Features.ShakeHighlightDelayMs)
	activeService = service
	defer func() {
		activeService = nil
	}()

	if err := service.installTray(instance); err != nil {
		return err
	}
	defer service.removeTray()

	manager, err := highlight.New(instance)
	if err == nil {
		service.Highlight = manager
		defer manager.Close()
	}

	hook, err := win32.SetWindowsHookEx(win32.WH_MOUSE_LL, hookProc, instance, 0)
	if err != nil {
		return err
	}
	defer win32.UnhookWindowsHookEx(hook)

	keyboardHook, err := win32.SetWindowsHookEx(win32.WH_KEYBOARD_LL, keyboardProc, instance, 0)
	if err != nil {
		return err
	}
	defer win32.UnhookWindowsHookEx(keyboardHook)

	service.Hook = hook
	service.KeyboardHook = keyboardHook
	go service.restoreSavedLayoutOnLaunch()
	return win32.RunMessageLoop()
}

func (s *Service) installTray(instance win32.HINSTANCE) error {
	class := win32.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win32.WNDCLASSEX{})),
		LpfnWndProc:   trayProc,
		HInstance:     instance,
		LpszClassName: syscall.StringToUTF16Ptr(trayClassName),
	}
	if _, err := win32.RegisterClassEx(&class); err != nil {
		// Re-registering the class in the same process is harmless.
	}

	hwnd, err := win32.CreateWindowEx(
		win32.WS_EX_TOOLWINDOW|win32.WS_EX_NOACTIVATE,
		trayClassName,
		"OJTools Tray",
		win32.WS_POPUP,
		0,
		0,
		0,
		0,
		0,
		0,
		instance,
		0,
	)
	if err != nil {
		return err
	}
	_ = win32.SetWindowPos(hwnd, 0, -32000, -32000, 0, 0, win32.SWP_NOSIZE|win32.SWP_NOZORDER|win32.SWP_NOACTIVATE)
	win32.ShowWindow(hwnd, win32.SW_HIDE)
	s.HWND = hwnd

	icon, err := createTrayIcon()
	if err != nil {
		return err
	}
	s.Icon = icon

	data := win32.NOTIFYICONDATA{
		CbSize:           uint32(unsafe.Sizeof(win32.NOTIFYICONDATA{})),
		HWnd:             hwnd,
		UID:              trayIconID,
		UFlags:           win32.NIF_MESSAGE | win32.NIF_ICON | win32.NIF_TIP,
		UCallbackMessage: trayCallbackMessage,
		HIcon:            icon,
	}
	win32.SetNotifyIconTip(&data, fmt.Sprintf("OJTools running (%s <-> %s)", s.Pair.Left.DeviceName, s.Pair.Right.DeviceName))
	if err := win32.ShellNotifyIcon(win32.NIM_ADD, &data); err != nil {
		win32.DestroyWindow(hwnd)
		s.HWND = 0
		return err
	}

	s.Tray = data
	return nil
}

func (s *Service) removeTray() {
	if s.Tray.HWnd != 0 {
		_ = win32.ShellNotifyIcon(win32.NIM_DELETE, &s.Tray)
		s.Tray.HWnd = 0
	}
	if s.Icon != 0 {
		win32.DestroyIcon(s.Icon)
		s.Icon = 0
	}
	if s.HWND != 0 {
		win32.DestroyWindow(s.HWND)
		s.HWND = 0
	}
}
