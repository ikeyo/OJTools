//go:build windows

package highlight

import (
	"math"
	"syscall"
	"time"
	"unsafe"

	"ojtools/internal/win32"
)

const (
	windowClassName   = "OJToolsShakeHighlight"
	windowSize        = int32(220)
	timerID           = uintptr(1)
	timerInterval     = uint32(10)
	appearDuration    = 220 * time.Millisecond
	sustainGrace      = 140 * time.Millisecond
	disappearDuration = 320 * time.Millisecond
)

var (
	activeManager = (*Manager)(nil)
	windowProc    = syscall.NewCallback(highlightWindowProc)
	chromaKey     = win32.RGB(255, 0, 255)
)

type Manager struct {
	HWND        win32.HWND
	Active      bool
	Visible     bool
	LastPos     win32.POINT
	TriggeredAt time.Time
	VisibleAt   time.Time
	LastShakeAt time.Time
	Delay       time.Duration
}

func New(instance win32.HINSTANCE) (*Manager, error) {
	cursor, err := win32.LoadCursor(win32.IDC_ARROW)
	if err != nil {
		return nil, err
	}

	class := win32.WNDCLASSEX{
		CbSize:        uint32(unsafe.Sizeof(win32.WNDCLASSEX{})),
		Style:         win32.CS_HREDRAW | win32.CS_VREDRAW,
		LpfnWndProc:   windowProc,
		HInstance:     instance,
		HCursor:       cursor,
		LpszClassName: syscall.StringToUTF16Ptr(windowClassName),
	}
	if _, err := win32.RegisterClassEx(&class); err != nil {
		// Re-registering in the same process is harmless.
	}

	hwnd, err := win32.CreateWindowEx(
		win32.WS_EX_TOPMOST|win32.WS_EX_TOOLWINDOW|win32.WS_EX_LAYERED|win32.WS_EX_TRANSPARENT|win32.WS_EX_NOACTIVATE,
		windowClassName,
		"OJTools Highlight",
		win32.WS_POPUP,
		0,
		0,
		windowSize,
		windowSize,
		0,
		0,
		instance,
		0,
	)
	if err != nil {
		return nil, err
	}

	if err := win32.SetLayeredWindowAttributes(hwnd, chromaKey, 0, win32.LWA_COLORKEY); err != nil {
		win32.DestroyWindow(hwnd)
		return nil, err
	}

	manager := &Manager{HWND: hwnd}
	activeManager = manager
	win32.ShowWindow(hwnd, win32.SW_HIDE)
	return manager, nil
}

func (m *Manager) Close() {
	if m == nil {
		return
	}
	m.Stop()
	if m.HWND != 0 {
		win32.DestroyWindow(m.HWND)
		m.HWND = 0
	}
	if activeManager == m {
		activeManager = nil
	}
}

func (m *Manager) Trigger(point win32.POINT, delay time.Duration) {
	if m == nil || m.HWND == 0 {
		return
	}
	now := time.Now()
	m.LastPos = point
	m.LastShakeAt = now
	m.Delay = delay

	if m.Active {
		return
	}

	m.Active = true
	m.Visible = false
	m.TriggeredAt = now
	m.VisibleAt = time.Time{}
	win32.SetTimer(m.HWND, timerID, timerInterval, 0)
	if delay <= 0 {
		m.showNow()
	} else {
		win32.ShowWindow(m.HWND, win32.SW_HIDE)
	}
}

func (m *Manager) Move(point win32.POINT) {
	if m == nil {
		return
	}
	m.LastPos = point
	if m.Active {
		m.position()
	}
}

func (m *Manager) Stop() {
	if m == nil || m.HWND == 0 {
		return
	}
	m.hide()
}

func highlightWindowProc(hwnd win32.HWND, msg uint32, wParam, lParam uintptr) uintptr {
	if activeManager == nil || activeManager.HWND != hwnd {
		return win32.DefWindowProc(hwnd, msg, wParam, lParam)
	}

	switch msg {
	case win32.WM_ERASEBKGND:
		return 1
	case win32.WM_NCHITTEST:
		return win32.HTTRANSPARENT
	case win32.WM_TIMER:
		if wParam == timerID {
			activeManager.tick()
			return 0
		}
	case win32.WM_PAINT:
		activeManager.paint()
		return 0
	case win32.WM_CLOSE:
		win32.ShowWindow(hwnd, win32.SW_HIDE)
		return 0
	case win32.WM_DESTROY:
		if activeManager != nil && activeManager.HWND == hwnd {
			activeManager.HWND = 0
			activeManager.Active = false
			activeManager = nil
		}
		return 0
	}

	return win32.DefWindowProc(hwnd, msg, wParam, lParam)
}

func (m *Manager) tick() {
	if !m.Active {
		win32.KillTimer(m.HWND, timerID)
		return
	}

	if !m.Visible {
		if time.Since(m.LastShakeAt) > sustainGrace {
			m.hide()
			return
		}
		if time.Since(m.TriggeredAt) >= m.Delay {
			m.showNow()
		}
		return
	}

	if time.Now().After(m.sustainUntil().Add(disappearDuration)) {
		m.hide()
		return
	}

	if err := win32.GetCursorPos(&m.LastPos); err == nil {
		m.position()
	}
	win32.InvalidateRect(m.HWND, nil, false)
}

func (m *Manager) hide() {
	m.Active = false
	m.Visible = false
	win32.KillTimer(m.HWND, timerID)
	win32.ShowWindow(m.HWND, win32.SW_HIDE)
}

func (m *Manager) showNow() {
	m.Visible = true
	m.VisibleAt = time.Now()
	m.position()
	win32.ShowWindow(m.HWND, win32.SW_SHOWNOACTIVATE)
	win32.InvalidateRect(m.HWND, nil, false)
}

func (m *Manager) position() {
	x := m.LastPos.X - windowSize/2
	y := m.LastPos.Y - windowSize/2
	_ = win32.SetWindowPos(m.HWND, 0, x, y, 0, 0, win32.SWP_NOSIZE|win32.SWP_NOZORDER|win32.SWP_NOACTIVATE|win32.SWP_SHOWWINDOW)
}

func (m *Manager) paint() {
	var ps win32.PAINTSTRUCT
	hdc, err := win32.BeginPaint(m.HWND, &ps)
	if err != nil {
		return
	}
	defer win32.EndPaint(m.HWND, &ps)

	var client win32.RECT
	if err := win32.GetClientRect(m.HWND, &client); err != nil {
		return
	}

	background, err := win32.CreateSolidBrush(chromaKey)
	if err == nil {
		defer win32.DeleteObject(win32.HGDIOBJ(background))
		_ = win32.FillRect(hdc, &client, background)
	}

	scale := animatedScale(time.Now(), m.VisibleAt, m.sustainUntil())
	centerX := client.Width() / 2
	centerY := client.Height() / 2

	outer := int32(math.Round((20 + 72) * scale))
	middle := int32(math.Round((12 + 48) * scale))
	inner := int32(math.Round((6 + 24) * scale))

	if outer < 12 {
		outer = 12
	}
	if middle < 8 {
		middle = 8
	}
	if inner < 4 {
		inner = 4
	}

	drawRing(hdc, centerX, centerY, outer, 5, win32.RGB(231, 104, 36))
	drawRing(hdc, centerX, centerY, middle, 4, win32.RGB(255, 197, 89))
	drawRing(hdc, centerX, centerY, inner, 3, win32.RGB(44, 165, 141))
	drawCenter(hdc, centerX, centerY, int32(math.Round(3+4*scale)), win32.RGB(28, 33, 42))
}

func drawRing(hdc win32.HDC, centerX, centerY, radius, width int32, color win32.COLORREF) {
	pen, err := win32.CreatePen(win32.PS_SOLID, width, color)
	if err != nil {
		return
	}
	defer win32.DeleteObject(pen)

	nullBrush := win32.GetStockObject(win32.NULL_BRUSH)
	prevPen := win32.SelectObject(hdc, pen)
	prevBrush := win32.SelectObject(hdc, nullBrush)
	defer func() {
		win32.SelectObject(hdc, prevPen)
		win32.SelectObject(hdc, prevBrush)
	}()

	win32.Ellipse(hdc, centerX-radius, centerY-radius, centerX+radius, centerY+radius)
}

func drawCenter(hdc win32.HDC, centerX, centerY, radius int32, color win32.COLORREF) {
	brush, err := win32.CreateSolidBrush(color)
	if err != nil {
		return
	}
	defer win32.DeleteObject(win32.HGDIOBJ(brush))

	pen, err := win32.CreatePen(win32.PS_SOLID, 1, color)
	if err != nil {
		return
	}
	defer win32.DeleteObject(pen)

	prevPen := win32.SelectObject(hdc, pen)
	prevBrush := win32.SelectObject(hdc, win32.HGDIOBJ(brush))
	defer func() {
		win32.SelectObject(hdc, prevPen)
		win32.SelectObject(hdc, prevBrush)
	}()

	win32.Ellipse(hdc, centerX-radius, centerY-radius, centerX+radius, centerY+radius)
}

func easeOut(t float64) float64 {
	inv := 1 - t
	return 1 - inv*inv*inv
}

func easeIn(t float64) float64 {
	return t * t * t
}

func (m *Manager) sustainUntil() time.Time {
	appearDone := m.VisibleAt.Add(appearDuration)
	shakeDone := m.LastShakeAt.Add(sustainGrace)
	if shakeDone.After(appearDone) {
		return shakeDone
	}
	return appearDone
}

func animatedScale(now, visibleAt, sustainUntil time.Time) float64 {
	if visibleAt.IsZero() {
		return 0
	}

	if now.Before(visibleAt) {
		return 0.18
	}

	if now.Before(visibleAt.Add(appearDuration)) {
		progress := float64(now.Sub(visibleAt)) / float64(appearDuration)
		return 0.18 + 0.82*easeOut(clamp01(progress))
	}

	if now.Before(sustainUntil) {
		return 1
	}

	shrink := float64(now.Sub(sustainUntil)) / float64(disappearDuration)
	return 1 - 0.82*easeIn(clamp01(shrink))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
