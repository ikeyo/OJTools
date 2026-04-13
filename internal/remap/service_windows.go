//go:build windows

package remap

import (
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
	"unsafe"

	"ojtools/internal/autostart"
	"ojtools/internal/calibration"
	"ojtools/internal/config"
	"ojtools/internal/highlight"
	"ojtools/internal/layout"
	"ojtools/internal/monitor"
	"ojtools/internal/win32"
)

var (
	activeService *Service
	hookProc      = syscall.NewCallback(lowLevelMouseProc)
	trayProc      = syscall.NewCallback(trayWindowProc)
)

const (
	trayClassName              = "OJToolsTrayWindow"
	trayCallbackMessage        = win32.WM_APP + 1
	trayIconID                 = 1
	trayCalibrationCommand     = 997
	trayResetCalibrationCommand = 998
	trayPauseCommand            = 999
	trayAutoStartCommand        = 1000
	trayRestoreOnLaunchCommand  = 1001
	trayShakeHighlightCommand   = 1002
	trayShakeDelay500Command    = 1003
	trayShakeDelay1000Command   = 1004
	trayShakeDelay2000Command   = 1005
	traySaveLayoutCommand       = 1006
	trayRestoreLayoutCommand    = 1007
	trayClearLayoutCommand      = 1008
	trayOpenConfigCommand       = 1009
	trayAboutCommand            = 1010
	trayExitCommand             = 1011

	shakeWindow      = 750 * time.Millisecond
	shakeMinSegment  = 10.0
	shakeMinPath     = 240.0
	shakeMaxNetRatio = 0.45
	shakeMinTurns    = 4
	shakeSuppressTTL = 150 * time.Millisecond
	fullscreenSlack  = int32(2)
)

type shakeSample struct {
	Point win32.POINT
	At    time.Time
}

type Service struct {
	Pair               monitor.Pair
	Transform          config.Transform
	ConfigPath         string
	Config             config.Config
	Hook               win32.HHOOK
	HWND               win32.HWND
	Tray               win32.NOTIFYICONDATA
	Icon               win32.HICON
	Prev               win32.POINT
	HavePrev           bool
	Paused             bool
	Highlight          *highlight.Manager
	ShakeSamples       []shakeSample
	ShakeSuppressUntil time.Time
	ShakeSuppressed    bool
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

	service.Hook = hook
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

func lowLevelMouseProc(nCode int32, wParam, lParam uintptr) uintptr {
	if activeService == nil {
		return 0
	}

	if nCode != win32.HC_ACTION {
		return win32.CallNextHookEx(activeService.Hook, nCode, wParam, lParam)
	}
	if uint32(wParam) != win32.WM_MOUSEMOVE {
		return win32.CallNextHookEx(activeService.Hook, nCode, wParam, lParam)
	}

	info := (*win32.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
	if info.Flags&win32.LLMHF_INJECTED != 0 {
		activeService.Prev = info.Pt
		activeService.HavePrev = true
		return win32.CallNextHookEx(activeService.Hook, nCode, wParam, lParam)
	}

	activeService.handleShakeMotion(info.Pt)

	if activeService.Paused {
		activeService.Prev = info.Pt
		activeService.HavePrev = true
		return win32.CallNextHookEx(activeService.Hook, nCode, wParam, lParam)
	}

	if !activeService.HavePrev {
		activeService.Prev = info.Pt
		activeService.HavePrev = true
		return win32.CallNextHookEx(activeService.Hook, nCode, wParam, lParam)
	}

	if mapped, ok := activeService.remap(activeService.Prev, info.Pt); ok {
		_ = win32.SetCursorPos(mapped.X, mapped.Y)
		activeService.Prev = mapped
		return 1
	}

	activeService.Prev = info.Pt
	return win32.CallNextHookEx(activeService.Hook, nCode, wParam, lParam)
}

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

func (s *Service) startCalibration() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath, "calibrate")
	cmd.Dir = filepath.Dir(exePath)
	return cmd.Start()
}

func (s *Service) resetCalibration() error {
	pairKey := monitor.PairKey(s.Pair)
	s.Config.Delete(pairKey)
	s.Transform = calibration.DefaultTransform(s.Pair)
	s.HavePrev = false
	return saveAppConfig(s.ConfigPath, s.Config)
}

func (s *Service) remap(prev, current win32.POINT) (win32.POINT, bool) {
	if crossingY, ok := s.Pair.CrossingFromLeftToRight(prev, current); ok {
		y := calibration.MapSecondaryToPrimary(s.Pair, s.Transform, crossingY)
		if s.Pair.PrimaryOnLeft() {
			y = calibration.MapPrimaryToSecondary(s.Pair, s.Transform, crossingY)
		}
		return win32.POINT{
			X: s.Pair.Right.Bounds.Left + 2,
			Y: clamp(int32(math.Round(y)), s.Pair.Right.Bounds.Top, s.Pair.Right.Bounds.Bottom-1),
		}, true
	}

	if crossingY, ok := s.Pair.CrossingFromRightToLeft(prev, current); ok {
		y := calibration.MapPrimaryToSecondary(s.Pair, s.Transform, crossingY)
		if s.Pair.PrimaryOnLeft() {
			y = calibration.MapSecondaryToPrimary(s.Pair, s.Transform, crossingY)
		}
		return win32.POINT{
			X: s.Pair.Left.Bounds.Right - 2,
			Y: clamp(int32(math.Round(y)), s.Pair.Left.Bounds.Top, s.Pair.Left.Bounds.Bottom-1),
		}, true
	}

	return win32.POINT{}, false
}

func (s *Service) handleShakeMotion(point win32.POINT) {
	if s.Highlight != nil {
		s.Highlight.Move(point)
	}
	if !s.Config.Features.ShakeCursorHighlight || s.Highlight == nil {
		s.resetShakeDetector(point, time.Now())
		return
	}

	now := time.Now()
	if s.shouldSuppressShake(now) {
		s.Highlight.Stop()
		s.resetShakeDetector(point, now)
		return
	}

	s.ShakeSamples = append(s.ShakeSamples, shakeSample{Point: point, At: now})
	s.trimShakeSamples(now)

	totalPath, netDistance, turns := s.shakeMetrics()
	if turns >= shakeMinTurns && totalPath >= shakeMinPath && netDistance <= totalPath*shakeMaxNetRatio {
		s.Highlight.Trigger(point, time.Duration(normalizeShakeDelay(s.Config.Features.ShakeHighlightDelayMs))*time.Millisecond)
	}
}

func (s *Service) trimShakeSamples(now time.Time) {
	keepFrom := 0
	for keepFrom < len(s.ShakeSamples) && now.Sub(s.ShakeSamples[keepFrom].At) > shakeWindow {
		keepFrom++
	}
	if keepFrom > 0 {
		s.ShakeSamples = append([]shakeSample(nil), s.ShakeSamples[keepFrom:]...)
	}
}

func (s *Service) shouldSuppressShake(now time.Time) bool {
	if now.Before(s.ShakeSuppressUntil) {
		return s.ShakeSuppressed
	}

	s.ShakeSuppressUntil = now.Add(shakeSuppressTTL)
	s.ShakeSuppressed = s.foregroundLooksFullscreen()
	return s.ShakeSuppressed
}

func (s *Service) foregroundLooksFullscreen() bool {
	hwnd := win32.GetForegroundWindow()
	if hwnd == 0 || hwnd == s.HWND {
		return false
	}

	var rect win32.RECT
	if err := win32.GetWindowRect(hwnd, &rect); err != nil {
		return false
	}

	monitorHandle := win32.MonitorFromWindow(hwnd, win32.MONITOR_DEFAULTTONEAREST)
	if monitorHandle == 0 {
		return false
	}

	var info win32.MONITORINFOEX
	if err := win32.GetMonitorInfo(monitorHandle, &info); err != nil {
		return false
	}

	monitorRect := info.RcMonitor
	return absInt32(rect.Left-monitorRect.Left) <= fullscreenSlack &&
		absInt32(rect.Top-monitorRect.Top) <= fullscreenSlack &&
		absInt32(rect.Right-monitorRect.Right) <= fullscreenSlack &&
		absInt32(rect.Bottom-monitorRect.Bottom) <= fullscreenSlack
}

func (s *Service) shakeMetrics() (float64, float64, int) {
	if len(s.ShakeSamples) < 3 {
		return 0, 0, 0
	}

	totalPath := 0.0
	turns := 0
	prevDX := 0.0
	prevDY := 0.0
	prevLen := 0.0
	havePrevVector := false

	for i := 1; i < len(s.ShakeSamples); i++ {
		dx := float64(s.ShakeSamples[i].Point.X - s.ShakeSamples[i-1].Point.X)
		dy := float64(s.ShakeSamples[i].Point.Y - s.ShakeSamples[i-1].Point.Y)
		segLen := math.Hypot(dx, dy)
		if segLen < shakeMinSegment {
			continue
		}

		totalPath += segLen
		if havePrevVector {
			dot := ((prevDX * dx) + (prevDY * dy)) / (prevLen * segLen)
			if dot < 0.35 {
				turns++
			}
		}

		prevDX = dx
		prevDY = dy
		prevLen = segLen
		havePrevVector = true
	}

	first := s.ShakeSamples[0].Point
	last := s.ShakeSamples[len(s.ShakeSamples)-1].Point
	netDistance := math.Hypot(float64(last.X-first.X), float64(last.Y-first.Y))
	return totalPath, netDistance, turns
}

func (s *Service) resetShakeDetector(point win32.POINT, now time.Time) {
	s.ShakeSamples = []shakeSample{{
		Point: point,
		At:    now,
	}}
}

func clamp(v, min, max int32) int32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func absInt32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func createTrayIcon() (win32.HICON, error) {
	exePath, err := os.Executable()
	if err == nil {
		icon, err := win32.ExtractSmallIcon(exePath)
		if err == nil {
			return icon, nil
		}
	}

	icon, err := win32.LoadIcon(win32.IDI_APPLICATION)
	if err == nil {
		return icon, nil
	}

	pattern := []string{
		"................",
		"................",
		"................",
		"................",
		"..#####..#####..",
		"..#www#..#www#..",
		"..#www#..#www#..",
		"..#www#..#www#..",
		"..#####..#####..",
		"................",
		"................",
		"................",
		"................",
		"................",
		"................",
		"................",
	}

	andBits := make([]byte, 32)
	xorBits := make([]byte, 32)
	for y, row := range pattern {
		for x, cell := range row {
			byteIndex := y*2 + x/8
			bit := byte(1 << (7 - (x % 8)))

			switch cell {
			case '.':
				andBits[byteIndex] |= bit
			case '#':
				// black pixel: and=0 xor=0
			case 'w':
				xorBits[byteIndex] |= bit
			default:
				andBits[byteIndex] |= bit
			}
		}
	}

	return win32.CreateIcon(16, 16, 1, 1, andBits, xorBits)
}

func normalizeShakeDelay(delayMs int) int {
	switch delayMs {
	case 500, 1000, 2000:
		return delayMs
	default:
		return 500
	}
}

func (s *Service) restoreSavedLayoutOnLaunch() {
	time.Sleep(4 * time.Second)
	_, _, _ = restoreWindowLayout(true)
}

func (s *Service) refreshTrayTip() {
	if s.Tray.HWnd == 0 {
		return
	}

	status := "running"
	if s.Paused {
		status = "paused"
	}

	data := s.Tray
	win32.SetNotifyIconTip(&data, fmt.Sprintf("OJTools %s (%s <-> %s)", status, s.Pair.Left.DeviceName, s.Pair.Right.DeviceName))
	if err := win32.ShellNotifyIcon(win32.NIM_MODIFY, &data); err == nil {
		s.Tray = data
	}
}

func saveWindowLayout() error {
	path, err := layout.DefaultPath()
	if err != nil {
		return err
	}
	_, err = layout.SaveSnapshot(path)
	return err
}

func restoreWindowLayout(clearPending bool) (bool, layout.RestoreResult, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return false, layout.RestoreResult{}, err
	}

	if clearPending {
		return layout.MaybeRestoreOnLaunch(path)
	}

	result, err := layout.RestoreSnapshot(path, false)
	return true, result, err
}

func clearWindowLayout() error {
	path, err := layout.DefaultPath()
	if err != nil {
		return err
	}
	return layout.Clear(path)
}

func windowLayoutStatus() (layout.SnapshotStatus, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return layout.SnapshotStatus{}, err
	}
	return layout.Status(path)
}

func setRestoreOnLaunch(enabled bool) (layout.SnapshotStatus, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return layout.SnapshotStatus{}, err
	}
	return layout.SetRestoreOnLaunch(path, enabled)
}

func openConfigFolder() error {
	path, err := layout.DefaultPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	cmd := exec.Command("explorer.exe", dir)
	cmd.Dir = dir
	return cmd.Start()
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
		"OJTools\n\nCursor remap: %s\nPair: %s <-> %s\nAuto-start: %s\nShake to find cursor: %s\nShake delay: %d ms\n%s\nConfig folder: %s",
		remapState,
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

func appConfigDir() (string, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

func saveAppConfig(path string, cfg config.Config) error {
	if path == "" {
		return nil
	}
	return config.Save(path, cfg)
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
