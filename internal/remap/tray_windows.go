//go:build windows

package remap

import (
	"fmt"

	"ojtools/internal/autostart"
	"ojtools/internal/layout"
	"ojtools/internal/win32"
)

func trayWindowProc(hwnd win32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if activeService == nil {
		return win32.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case trayCallbackMessage:
		switch uint32(lParam) {
		case win32.WM_LBUTTONDBLCLK:
			if err := activeService.startCalibration(); err == nil {
				win32.DestroyWindow(hwnd)
			}
			return 0
		case win32.WM_RBUTTONUP:
			return activeService.showTrayMenu(hwnd)
		}
	case win32.WM_CLOSE:
		win32.DestroyWindow(hwnd)
		return 0
	case win32.WM_DESTROY:
		win32.PostQuitMessage(0)
		return 0
	}

	return win32.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (s *Service) showTrayMenu(hwnd win32.HWND) uintptr {
	menu, err := win32.CreatePopupMenu()
	if err != nil {
		return 0
	}
	defer win32.DestroyMenu(menu)

	autoStartEnabled, err := autostart.IsEnabled()
	if err != nil {
		autoStartEnabled = false
	}

	layoutStatus, err := windowLayoutStatus()
	if err != nil {
		layoutStatus = layout.SnapshotStatus{}
	}
	hasLayout := layoutStatus.Exists && layoutStatus.WindowCount > 0

	if err := appendMenuItem(menu, trayCalibrationCommand, "Cursor Calibration...", false, false); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayResetCalibrationCommand, "Reset DPI Calibration", false, false); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayPauseCommand, "Pause Cursor Remap", s.Paused, false); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayLockKeyboardCommand, fmt.Sprintf("Lock Keyboard (%s)", keyboardLockHotkey), s.KeyboardLocked, false); err != nil {
		return 0
	}
	if err := appendMenuSeparator(menu); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayBlockedKeyStatusCommand, s.blockedKeyStatusLabel(), false, true); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayToggleBlockedKeyCommand, s.blockedKeyToggleLabel(), s.Config.Features.BlockedKeyEnabled, !s.hasBlockedKey()); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayCaptureBlockedKeyCommand, s.blockedKeyCaptureLabel(), false, false); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayClearBlockedKeyCommand, "Clear Blocked Key", false, !s.hasBlockedKey()); err != nil {
		return 0
	}
	if err := appendMenuSeparator(menu); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayAutoStartCommand, "Auto-Start", autoStartEnabled, false); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayRestoreOnLaunchCommand, "Restore Layout On Launch", hasLayout && layoutStatus.RestoreOnLaunch, !hasLayout); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayShakeHighlightCommand, "Shake To Find Cursor", s.Config.Features.ShakeCursorHighlight, false); err != nil {
		return 0
	}
	delayDisabled := !s.Config.Features.ShakeCursorHighlight
	if err := appendMenuItem(menu, trayShakeDelay500Command, "Shake Delay: 0.5 s", s.Config.Features.ShakeHighlightDelayMs == 500, delayDisabled); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayShakeDelay1000Command, "Shake Delay: 1 s", s.Config.Features.ShakeHighlightDelayMs == 1000, delayDisabled); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayShakeDelay2000Command, "Shake Delay: 2 s", s.Config.Features.ShakeHighlightDelayMs == 2000, delayDisabled); err != nil {
		return 0
	}
	if err := appendMenuSeparator(menu); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, traySaveLayoutCommand, "Save Window Layout", false, false); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayRestoreLayoutCommand, "Restore Window Layout", false, !hasLayout); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayClearLayoutCommand, "Clear Saved Window Layout", false, !hasLayout); err != nil {
		return 0
	}
	if err := appendMenuSeparator(menu); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayOpenConfigCommand, "Open Config Folder", false, false); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayAboutCommand, "About", false, false); err != nil {
		return 0
	}
	if err := appendMenuSeparator(menu); err != nil {
		return 0
	}
	if err := appendMenuItem(menu, trayExitCommand, "Exit OJTools", false, false); err != nil {
		return 0
	}

	var cursor win32.POINT
	if err := win32.GetCursorPos(&cursor); err != nil {
		return 0
	}

	win32.SetForegroundWindow(hwnd)
	cmd := win32.TrackPopupMenu(menu, win32.TPM_RETURNCMD|win32.TPM_RIGHTBUTTON, cursor.X, cursor.Y, 0, hwnd, nil)

	switch cmd {
	case trayCalibrationCommand:
		if err := s.startCalibration(); err == nil {
			win32.DestroyWindow(hwnd)
		}
	case trayResetCalibrationCommand:
		_ = s.resetCalibration()
	case trayPauseCommand:
		s.Paused = !s.Paused
		s.HavePrev = false
		s.refreshTrayTip()
	case trayLockKeyboardCommand:
		s.toggleKeyboardLock()
	case trayToggleBlockedKeyCommand:
		s.toggleBlockedKey()
	case trayCaptureBlockedKeyCommand:
		s.beginBlockedKeyCapture()
	case trayClearBlockedKeyCommand:
		s.clearBlockedKey()
	case trayAutoStartCommand:
		if autoStartEnabled {
			_ = autostart.Disable()
		} else {
			_ = autostart.EnableCurrentExecutable()
		}
	case trayRestoreOnLaunchCommand:
		if hasLayout {
			_, _ = setRestoreOnLaunch(!layoutStatus.RestoreOnLaunch)
		}
	case trayShakeHighlightCommand:
		s.Config.Features.ShakeCursorHighlight = !s.Config.Features.ShakeCursorHighlight
		s.Config.Features.ShakeHighlightDelayMs = normalizeShakeDelay(s.Config.Features.ShakeHighlightDelayMs)
		_ = saveAppConfig(s.ConfigPath, s.Config)
		if !s.Config.Features.ShakeCursorHighlight && s.Highlight != nil {
			s.Highlight.Stop()
		}
	case trayShakeDelay500Command:
		s.Config.Features.ShakeHighlightDelayMs = 500
		_ = saveAppConfig(s.ConfigPath, s.Config)
	case trayShakeDelay1000Command:
		s.Config.Features.ShakeHighlightDelayMs = 1000
		_ = saveAppConfig(s.ConfigPath, s.Config)
	case trayShakeDelay2000Command:
		s.Config.Features.ShakeHighlightDelayMs = 2000
		_ = saveAppConfig(s.ConfigPath, s.Config)
	case traySaveLayoutCommand:
		go saveWindowLayout()
	case trayRestoreLayoutCommand:
		if hasLayout {
			go restoreWindowLayout(false)
		}
	case trayClearLayoutCommand:
		if hasLayout {
			_ = clearWindowLayout()
		}
	case trayOpenConfigCommand:
		_ = openConfigFolder()
	case trayAboutCommand:
		_ = s.showAbout()
	case trayExitCommand:
		win32.DestroyWindow(hwnd)
	}

	return 0
}

func (s *Service) refreshTrayTip() {
	if s.Tray.HWnd == 0 {
		return
	}

	status := "running"
	if s.Paused {
		status = "paused"
	}
	keyboard := "keyboard unlocked"
	if s.KeyboardLocked {
		keyboard = "keyboard locked"
	}
	blockedKey := s.blockedKeySummary()

	data := s.Tray
	win32.SetNotifyIconTip(&data, fmt.Sprintf("OJTools %s, %s, %s (%s <-> %s)", status, keyboard, blockedKey, s.Pair.Left.DeviceName, s.Pair.Right.DeviceName))
	if err := win32.ShellNotifyIcon(win32.NIM_MODIFY, &data); err == nil {
		s.Tray = data
	}
}

func (s *Service) showAbout() error {
	autoStartEnabled, _ := autostart.IsEnabled()
	layoutStatus, _ := windowLayoutStatus()
	configDir, err := appConfigDir()
	if err != nil {
		configDir = "Unavailable"
	}

	remapState := "Active"
	if s.Paused {
		remapState = "Paused"
	}
	keyboardState := "Unlocked"
	if s.KeyboardLocked {
		keyboardState = "Locked"
	}
	blockedKeyState := "None"
	if s.hasBlockedKey() {
		blockedKeyState = vkName(s.Config.Features.BlockedVK)
		if s.Config.Features.BlockedKeyEnabled {
			blockedKeyState += " (On)"
		} else {
			blockedKeyState += " (Off)"
		}
	}
	if s.CaptureNextBlockedKey {
		blockedKeyState = "Waiting for next key..."
	}

	layoutSummary := "No saved window layout"
	if layoutStatus.Exists {
		restoreState := "Off"
		if layoutStatus.RestoreOnLaunch {
			restoreState = "On"
		}
		layoutSummary = fmt.Sprintf("Saved window layout: %d windows\nRestore on launch: %s", layoutStatus.WindowCount, restoreState)
	}

	autoStartState := "Off"
	if autoStartEnabled {
		autoStartState = "On"
	}

	shakeState := "Off"
	if s.Config.Features.ShakeCursorHighlight {
		shakeState = "On"
	}
	shakeDelay := normalizeShakeDelay(s.Config.Features.ShakeHighlightDelayMs)

	message := fmt.Sprintf(
		"OJTools\n\nCursor remap: %s\nKeyboard lock: %s (%s)\nBlocked key: %s\nPair: %s <-> %s\nAuto-start: %s\nShake to find cursor: %s\nShake delay: %d ms\n%s\nConfig folder: %s",
		remapState,
		keyboardState,
		keyboardLockHotkey,
		blockedKeyState,
		s.Pair.Left.DeviceName,
		s.Pair.Right.DeviceName,
		autoStartState,
		shakeState,
		shakeDelay,
		layoutSummary,
		configDir,
	)
	return win32.MessageBox(s.HWND, message, "About OJTools", win32.MB_OK)
}

func appendMenuItem(menu win32.HMENU, command uintptr, label string, checked bool, disabled bool) error {
	flags := uint32(win32.MF_STRING)
	if checked {
		flags |= win32.MF_CHECKED
	}
	if disabled {
		flags |= win32.MF_GRAYED
	}
	return win32.AppendMenu(menu, flags, command, label)
}

func appendMenuSeparator(menu win32.HMENU) error {
	return win32.AppendMenu(menu, win32.MF_SEPARATOR, 0, "")
}
