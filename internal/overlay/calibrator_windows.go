//go:build windows

package overlay

import (
	"fmt"
	"math"
	"runtime"
	"syscall"
	"unsafe"

	"ojtools/internal/calibration"
	"ojtools/internal/config"
	"ojtools/internal/monitor"
	"ojtools/internal/win32"
)

const className = "OJToolsCalibrationOverlay"

var (
	activeOverlay *Session
	windowProc    = syscall.NewCallback(overlayWndProc)
	hookProc      = syscall.NewCallback(overlayMouseProc)
)

type SaveFunc func(config.Transform) error

type Result struct {
	Transform config.Transform
	Saved     bool
}

type Session struct {
	Pair             monitor.Pair
	Transform        config.Transform
	DefaultTransform config.Transform
	Save             SaveFunc
	Saved            bool
	SaveErr          error
	Hook             win32.HHOOK
	Prev             win32.POINT
	HavePrev         bool
	HWND             win32.HWND
	Top              int32
	Height           int32
	Dragging         bool
	DragStartY       int32
	DragStartOffset  float64
}

func Run(pair monitor.Pair, transform, defaultTransform config.Transform, save SaveFunc) (Result, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	bounds := pair.Left.Bounds
	if pair.Right.Bounds.Top < bounds.Top {
		bounds.Top = pair.Right.Bounds.Top
	}
	if pair.Right.Bounds.Bottom > bounds.Bottom {
		bounds.Bottom = pair.Right.Bounds.Bottom
	}

	if bounds.Height() < 480 {
		padding := int32((480 - bounds.Height()) / 2)
		bounds.Top -= padding
		bounds.Bottom += padding
	}

	instance, err := win32.GetModuleHandle()
	if err != nil {
		return Result{}, err
	}
	cursor, err := win32.LoadCursor(win32.IDC_ARROW)
	if err != nil {
		return Result{}, err
	}

	class := win32.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win32.WNDCLASSEX{})),
		Style:         win32.CS_HREDRAW | win32.CS_VREDRAW,
		LpfnWndProc:   windowProc,
		HInstance:     instance,
		HCursor:       cursor,
		LpszClassName: syscall.StringToUTF16Ptr(className),
	}
	if _, err := win32.RegisterClassEx(&class); err != nil {
		// Windows may report that the class already exists if calibrate is re-entered.
	}

	width := int32(380)
	x := pair.BoundaryX - width/2

	activeOverlay = &Session{
		Pair:             pair,
		Transform:        clampTransform(transform),
		DefaultTransform: clampTransform(defaultTransform),
		Save:             save,
		Top:              bounds.Top,
		Height:           bounds.Height(),
	}

	hook, err := win32.SetWindowsHookEx(win32.WH_MOUSE_LL, hookProc, instance, 0)
	if err != nil {
		activeOverlay = nil
		return Result{}, err
	}
	activeOverlay.Hook = hook
	defer func() {
		if hook != 0 {
			win32.UnhookWindowsHookEx(hook)
		}
	}()

	hwnd, err := win32.CreateWindowEx(
		win32.WS_EX_TOPMOST|win32.WS_EX_TOOLWINDOW,
		className,
		"OJTools Calibration",
		win32.WS_POPUP|win32.WS_VISIBLE,
		x,
		bounds.Top,
		width,
		bounds.Height(),
		0,
		0,
		instance,
		0,
	)
	if err != nil {
		return Result{}, err
	}

	activeOverlay.HWND = hwnd
	win32.ShowWindow(hwnd, win32.SW_SHOW)
	if err := win32.UpdateWindow(hwnd); err != nil {
		return Result{}, err
	}

	if err := win32.RunMessageLoop(); err != nil {
		return Result{}, err
	}
	if activeOverlay == nil {
		return Result{}, nil
	}
	result := Result{
		Transform: activeOverlay.Transform,
		Saved:     activeOverlay.Saved,
	}
	err = activeOverlay.SaveErr
	activeOverlay = nil
	return result, err
}

func overlayMouseProc(nCode int32, wParam, lParam uintptr) uintptr {
	if activeOverlay == nil {
		return 0
	}

	if nCode != win32.HC_ACTION {
		return win32.CallNextHookEx(activeOverlay.Hook, nCode, wParam, lParam)
	}
	if uint32(wParam) != win32.WM_MOUSEMOVE {
		return win32.CallNextHookEx(activeOverlay.Hook, nCode, wParam, lParam)
	}

	info := (*win32.MSLLHOOKSTRUCT)(unsafe.Pointer(lParam))
	if info.Flags&win32.LLMHF_INJECTED != 0 {
		activeOverlay.Prev = info.Pt
		activeOverlay.HavePrev = true
		return win32.CallNextHookEx(activeOverlay.Hook, nCode, wParam, lParam)
	}

	if !activeOverlay.HavePrev {
		activeOverlay.Prev = info.Pt
		activeOverlay.HavePrev = true
		return win32.CallNextHookEx(activeOverlay.Hook, nCode, wParam, lParam)
	}

	if mapped, ok := activeOverlay.remap(activeOverlay.Prev, info.Pt); ok {
		_ = win32.SetCursorPos(mapped.X, mapped.Y)
		activeOverlay.Prev = mapped
		return 1
	}

	activeOverlay.Prev = info.Pt
	return win32.CallNextHookEx(activeOverlay.Hook, nCode, wParam, lParam)
}

func overlayWndProc(hwnd win32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if activeOverlay == nil {
		return win32.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case win32.WM_ERASEBKGND:
		return 1
	case win32.WM_PAINT:
		activeOverlay.paint()
		return 0
	case win32.WM_MOUSEWHEEL:
		delta := int32(win32.SignedHiWord(wParam))
		step := 0.02
		if win32.LoWord(wParam)&win32.MK_SHIFT != 0 {
			step = 0.005
		}
		if delta > 0 {
			activeOverlay.Transform.Scale += step
		} else if delta < 0 {
			activeOverlay.Transform.Scale -= step
		}
		activeOverlay.Transform = clampTransform(activeOverlay.Transform)
		win32.InvalidateRect(hwnd, nil, false)
		return 0
	case win32.WM_LBUTTONDOWN:
		activeOverlay.Dragging = true
		activeOverlay.DragStartY = int32(win32.SignedHiWord(lParam))
		activeOverlay.DragStartOffset = activeOverlay.Transform.Offset
		win32.SetCapture(hwnd)
		return 0
	case win32.WM_MOUSEMOVE:
		if activeOverlay.Dragging {
			currentY := int32(win32.SignedHiWord(lParam))
			delta := currentY - activeOverlay.DragStartY
			activeOverlay.Transform.Offset = activeOverlay.DragStartOffset + float64(delta)
			activeOverlay.Transform = clampTransform(activeOverlay.Transform)
			win32.InvalidateRect(hwnd, nil, false)
		}
		return 0
	case win32.WM_LBUTTONUP:
		activeOverlay.Dragging = false
		win32.ReleaseCapture()
		return 0
	case win32.WM_KEYDOWN:
		switch wParam {
		case win32.VK_ESCAPE:
			win32.DestroyWindow(hwnd)
		case win32.VK_RETURN, 'S':
			if activeOverlay.Save != nil {
				activeOverlay.SaveErr = activeOverlay.Save(activeOverlay.Transform)
				if activeOverlay.SaveErr != nil {
					return 0
				}
			}
			activeOverlay.Saved = true
			win32.DestroyWindow(hwnd)
		case 'R':
			activeOverlay.Transform = activeOverlay.DefaultTransform
			win32.InvalidateRect(hwnd, nil, false)
		case win32.VK_UP:
			activeOverlay.Transform.Offset -= 2
			win32.InvalidateRect(hwnd, nil, false)
		case win32.VK_DOWN:
			activeOverlay.Transform.Offset += 2
			win32.InvalidateRect(hwnd, nil, false)
		case win32.VK_LEFT, win32.VK_SUBTRACT, win32.VK_OEM_MINUS:
			activeOverlay.Transform.Scale -= 0.01
			activeOverlay.Transform = clampTransform(activeOverlay.Transform)
			win32.InvalidateRect(hwnd, nil, false)
		case win32.VK_RIGHT, win32.VK_ADD, win32.VK_OEM_PLUS:
			activeOverlay.Transform.Scale += 0.01
			activeOverlay.Transform = clampTransform(activeOverlay.Transform)
			win32.InvalidateRect(hwnd, nil, false)
		}
		return 0
	case win32.WM_CLOSE:
		win32.DestroyWindow(hwnd)
		return 0
	case win32.WM_DESTROY:
		win32.PostQuitMessage(0)
		return 0
	default:
		return win32.DefWindowProc(hwnd, msg, wParam, lParam)
	}
}

func (s *Session) paint() {
	var ps win32.PAINTSTRUCT
	hdc, err := win32.BeginPaint(s.HWND, &ps)
	if err != nil {
		return
	}
	defer win32.EndPaint(s.HWND, &ps)

	var client win32.RECT
	if err := win32.GetClientRect(s.HWND, &client); err != nil {
		return
	}

	background, err := win32.CreateSolidBrush(win32.RGB(248, 246, 238))
	if err == nil {
		defer win32.DeleteObject(win32.HGDIOBJ(background))
		_ = win32.FillRect(hdc, &client, background)
	}

	win32.SetBkMode(hdc, win32.BKMODE_TRANSPARENT)
	win32.SetTextColor(hdc, win32.RGB(28, 33, 42))

	leftX := int32(92)
	centerX := client.Width() / 2
	rightX := client.Width() - 92
	primary := s.Pair.PrimaryMonitor()
	secondary := s.Pair.SecondaryMonitor()
	primaryX := rightX
	secondaryX := leftX
	if s.Pair.PrimaryOnLeft() {
		primaryX = leftX
		secondaryX = rightX
	}
	secondaryStartValue := s.primaryValueForGlobalY(float64(secondary.Bounds.Top))
	secondaryEndValue := s.primaryValueForGlobalY(float64(secondary.Bounds.Bottom))

	s.drawBand(hdc, leftX, 16, client.Height()-16)
	s.drawBand(hdc, rightX, 16, client.Height()-16)

	win32.MoveToEx(hdc, centerX, 0)
	win32.LineTo(hdc, centerX, client.Height())

	leftTop := s.Pair.Left.Bounds.Top - s.Top
	leftBottom := s.Pair.Left.Bounds.Bottom - s.Top
	rightTop := s.Pair.Right.Bounds.Top - s.Top
	rightBottom := s.Pair.Right.Bounds.Bottom - s.Top

	s.drawMonitorCaps(hdc, leftX, leftTop, leftBottom)
	s.drawMonitorCaps(hdc, rightX, rightTop, rightBottom)

	s.drawText(hdc, 14, 14, fmt.Sprintf("Left  %s  %dx%d  %.0f%%  %d DPI", s.Pair.Left.DeviceName, s.Pair.Left.Bounds.Width(), s.Pair.Left.Bounds.Height(), s.Pair.Left.ScaleY*100, s.Pair.Left.DpiY))
	s.drawText(hdc, 14, 34, fmt.Sprintf("Right %s  %dx%d  %.0f%%  %d DPI", s.Pair.Right.DeviceName, s.Pair.Right.Bounds.Width(), s.Pair.Right.Bounds.Height(), s.Pair.Right.ScaleY*100, s.Pair.Right.DpiY))
	s.drawText(hdc, 14, 58, fmt.Sprintf("Primary -> Secondary  %s -> %s", primary.DeviceName, secondary.DeviceName))
	s.drawText(hdc, 14, 78, fmt.Sprintf("Secondary starts at primary %.2f and ends at %.2f", secondaryStartValue, secondaryEndValue))
	s.drawText(hdc, 14, 98, fmt.Sprintf("Scale %.4f   Offset %.1f px", s.Transform.Scale, s.Transform.Offset))
	s.drawText(hdc, 14, 118, "Wheel / Left-Right: scale   Drag / Up-Down: offset   Enter/S: save   R: reset   Esc: close")

	baseStep := float64(primary.Bounds.Height()) / 100.0
	if baseStep < 3 {
		baseStep = 3
	}

	for i := 0; i <= 100; i++ {
		primaryYGlobal := float64(primary.Bounds.Top) + (float64(i) * baseStep)
		primaryY := int32(math.Round(primaryYGlobal)) - s.Top

		s.drawTick(hdc, primaryX, primaryY, i)
	}

	startTick := int(math.Ceil(secondaryStartValue))
	endTick := int(math.Floor(secondaryEndValue))
	if startTick > endTick {
		startTick, endTick = endTick, startTick
	}

	for value := startTick; value <= endTick; value++ {
		primaryYGlobal := s.primaryGlobalYForValue(float64(value))
		secondaryY := int32(math.Round(calibration.MapPrimaryToSecondary(s.Pair, s.Transform, primaryYGlobal))) - s.Top
		label := value%10 == 0 || value == startTick
		s.drawTickValue(hdc, secondaryX, secondaryY, value, label)
	}
}

func (s *Session) drawBand(hdc win32.HDC, x, top, bottom int32) {
	band := win32.RECT{
		Left:   x - 38,
		Top:    top,
		Right:  x + 38,
		Bottom: bottom,
	}
	brush, err := win32.CreateSolidBrush(win32.RGB(233, 230, 220))
	if err == nil {
		defer win32.DeleteObject(win32.HGDIOBJ(brush))
		_ = win32.FillRect(hdc, &band, brush)
	}
}

func (s *Session) drawMonitorCaps(hdc win32.HDC, x, top, bottom int32) {
	win32.MoveToEx(hdc, x-48, top)
	win32.LineTo(hdc, x+48, top)
	win32.MoveToEx(hdc, x-48, bottom)
	win32.LineTo(hdc, x+48, bottom)
}

func (s *Session) drawTick(hdc win32.HDC, x, y int32, index int) {
	s.drawTickValue(hdc, x, y, index, index%10 == 0)
}

func (s *Session) drawTickValue(hdc win32.HDC, x, y int32, value int, label bool) {
	if y < 118 || y > s.Height-16 {
		return
	}

	length := int32(10)
	if value%10 == 0 {
		length = 26
	} else if value%5 == 0 {
		length = 18
	}

	win32.MoveToEx(hdc, x-length, y)
	win32.LineTo(hdc, x+length, y)

	if label {
		s.drawText(hdc, x+34, y-8, fmt.Sprintf("%d", value))
	}
}

func (s *Session) drawText(hdc win32.HDC, x, y int32, text string) {
	_ = win32.TextOut(hdc, x, y, text)
}

func (s *Session) remap(prev, current win32.POINT) (win32.POINT, bool) {
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

func clamp(v, min, max int32) int32 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func clampTransform(transform config.Transform) config.Transform {
	if transform.Scale < 0.2 {
		transform.Scale = 0.2
	}
	if transform.Scale > 5 {
		transform.Scale = 5
	}
	if math.IsNaN(transform.Scale) || math.IsInf(transform.Scale, 0) {
		transform.Scale = 1
	}
	if math.IsNaN(transform.Offset) || math.IsInf(transform.Offset, 0) {
		transform.Offset = 0
	}
	return transform
}

func (s *Session) primaryValueForGlobalY(y float64) float64 {
	primary := s.Pair.PrimaryMonitor()
	height := float64(primary.Bounds.Height())
	if height == 0 {
		return 0
	}

	return ((y - float64(primary.Bounds.Top)) / height) * 100.0
}

func (s *Session) primaryGlobalYForValue(value float64) float64 {
	primary := s.Pair.PrimaryMonitor()
	return float64(primary.Bounds.Top) + (float64(primary.Bounds.Height()) * value / 100.0)
}
