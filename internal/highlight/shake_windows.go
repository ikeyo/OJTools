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
	windowClassName    = "OJToolsShakeHighlight"
	initialWindowSize  = int32(520)
	minTrailWindowSize = int32(96)
	trailWindowPadding = int32(32)
	timerID            = uintptr(1)
	timerInterval      = uint32(10)
	sustainGrace       = 140 * time.Millisecond
	trailMinDistance   = 4.0
	trailTipWidth      = 1.5
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
	Settings    Settings
	Trail       []trailSample
	GDIPlus     uintptr
}

type Settings struct {
	Color       win32.COLORREF
	HeadWidth   float64
	MaxDistance float64
	Fade        time.Duration
}

type trailSample struct {
	Point win32.POINT
	At    time.Time
}

type trailFrame struct {
	X      int32
	Y      int32
	Width  int32
	Height int32
	Points []win32.POINT
}

func New(instance win32.HINSTANCE, settings Settings) (*Manager, error) {
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
		initialWindowSize,
		initialWindowSize,
		0,
		0,
		instance,
		0,
	)
	if err != nil {
		return nil, err
	}

	token, err := win32.GDIPlusStartup()
	if err != nil {
		win32.DestroyWindow(hwnd)
		return nil, err
	}

	manager := &Manager{
		HWND:     hwnd,
		Settings: normalizeSettings(settings),
		GDIPlus:  token,
	}
	activeManager = manager
	win32.ShowWindow(hwnd, win32.SW_HIDE)
	return manager, nil
}

func (m *Manager) SetSettings(settings Settings) {
	if m == nil {
		return
	}
	m.Settings = normalizeSettings(settings)
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
	win32.GDIPlusShutdown(m.GDIPlus)
	m.GDIPlus = 0
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
	m.addTrailPoint(point, now)

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
		m.addTrailPoint(point, time.Now())
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

	now := time.Now()
	if !m.Visible {
		if now.Sub(m.LastShakeAt) > sustainGrace {
			m.hide()
			return
		}
		if now.Sub(m.TriggeredAt) >= m.Delay {
			m.showNow()
		}
		return
	}

	m.trimTrail(now)
	if now.After(m.sustainUntil()) {
		m.hide()
		return
	}

	if err := win32.GetCursorPos(&m.LastPos); err == nil {
		m.addTrailPoint(m.LastPos, now)
	}
	m.render()
}

func (m *Manager) hide() {
	m.Active = false
	m.Visible = false
	m.Trail = nil
	win32.KillTimer(m.HWND, timerID)
	win32.ShowWindow(m.HWND, win32.SW_HIDE)
}

func (m *Manager) showNow() {
	m.Visible = true
	m.VisibleAt = time.Now()
	m.render()
	win32.ShowWindow(m.HWND, win32.SW_SHOWNOACTIVATE)
}

func (m *Manager) position() {
	x := m.LastPos.X - initialWindowSize/2
	y := m.LastPos.Y - initialWindowSize/2
	_ = win32.SetWindowPos(m.HWND, 0, x, y, 0, 0, win32.SWP_NOSIZE|win32.SWP_NOZORDER|win32.SWP_NOACTIVATE|win32.SWP_SHOWWINDOW)
}

func (m *Manager) paint() {
	var ps win32.PAINTSTRUCT
	_, err := win32.BeginPaint(m.HWND, &ps)
	if err != nil {
		return
	}
	defer win32.EndPaint(m.HWND, &ps)
}

func (m *Manager) render() {
	if m == nil || m.HWND == 0 {
		return
	}

	frame := m.trailFrame(time.Now())

	hdc, err := win32.CreateCompatibleDC(0)
	if err != nil {
		return
	}
	defer win32.DeleteDC(hdc)

	bitmap, bits, err := win32.CreateDIBSection(frame.Width, frame.Height)
	if err != nil {
		return
	}
	defer win32.DeleteObject(win32.HGDIOBJ(bitmap))

	previous := win32.SelectObject(hdc, win32.HGDIOBJ(bitmap))
	defer win32.SelectObject(hdc, previous)

	drawTrail(hdc, frame.Points, m.Settings.HeadWidth, m.Settings.Color)
	normalizeTrailPixels(bits, frame.Width, frame.Height, m.Settings.Color)
	_ = win32.UpdateLayeredWindowAlpha(m.HWND, frame.X, frame.Y, frame.Width, frame.Height, hdc)
}

func (m *Manager) addTrailPoint(point win32.POINT, now time.Time) {
	if len(m.Trail) > 0 {
		prev := m.Trail[len(m.Trail)-1].Point
		if math.Hypot(float64(point.X-prev.X), float64(point.Y-prev.Y)) < trailMinDistance {
			return
		}
	}

	m.Trail = append(m.Trail, trailSample{Point: point, At: now})
	m.trimTrail(now)
}

func (m *Manager) trimTrail(now time.Time) {
	keepFrom := 0
	for keepFrom < len(m.Trail) && now.Sub(m.Trail[keepFrom].At) > m.Settings.Fade {
		keepFrom++
	}
	if keepFrom > 0 {
		m.Trail = append([]trailSample(nil), m.Trail[keepFrom:]...)
	}
	m.trimTrailDistance()
}

func (m *Manager) trimTrailDistance() {
	if len(m.Trail) < 2 {
		return
	}

	total := 0.0
	keepFrom := len(m.Trail) - 1
	for keepFrom > 0 {
		prev := m.Trail[keepFrom-1]
		next := m.Trail[keepFrom]
		segment := pointDistance(prev.Point, next.Point)
		if segment == 0 {
			keepFrom--
			continue
		}
		if total+segment > m.Settings.MaxDistance {
			remaining := m.Settings.MaxDistance - total
			tail := interpolateTrailSample(prev, next, 1-(remaining/segment))
			m.Trail = append([]trailSample{tail}, m.Trail[keepFrom:]...)
			return
		}
		total += segment
		keepFrom--
	}

	if keepFrom > 0 {
		m.Trail = append([]trailSample(nil), m.Trail[keepFrom:]...)
	}
}

func (m *Manager) trailFrame(now time.Time) trailFrame {
	points := make([]win32.POINT, 0, len(m.Trail)+1)
	for _, sample := range m.Trail {
		if now.Sub(sample.At) <= m.Settings.Fade {
			points = append(points, sample.Point)
		}
	}
	if len(points) == 0 {
		points = append(points, m.LastPos)
	}

	left, top, right, bottom := trailBounds(points)
	padding := trailWindowPadding + int32(math.Ceil(m.Settings.HeadWidth))
	left -= padding
	top -= padding
	right += padding
	bottom += padding

	width := right - left
	height := bottom - top
	if width < minTrailWindowSize {
		center := (left + right) / 2
		left = center - minTrailWindowSize/2
		width = minTrailWindowSize
	}
	if height < minTrailWindowSize {
		center := (top + bottom) / 2
		top = center - minTrailWindowSize/2
		height = minTrailWindowSize
	}

	local := make([]win32.POINT, 0, len(points))
	for _, point := range points {
		local = append(local, win32.POINT{
			X: point.X - left,
			Y: point.Y - top,
		})
	}

	return trailFrame{
		X:      left,
		Y:      top,
		Width:  width,
		Height: height,
		Points: local,
	}
}

func drawTrail(hdc win32.HDC, points []win32.POINT, headWidth float64, color win32.COLORREF) {
	if len(points) < 2 {
		return
	}

	distances := cumulativeDistances(points)
	total := distances[len(distances)-1]
	if total <= 0 {
		return
	}

	argb := colorRefToARGB(color)
	for i := 1; i < len(points); i++ {
		if pointDistance(points[i-1], points[i]) == 0 {
			continue
		}
		previousWidth := trailWidthAt(distances[i-1]/total, headWidth)
		currentWidth := trailWidthAt(distances[i]/total, headWidth)
		polygon := trailSegmentPolygon(points[i-1], points[i], previousWidth, currentWidth)
		_ = win32.GDIPlusFillPolygon(hdc, polygon, argb)
	}

	for i := 1; i < len(points)-1; i++ {
		radius := int32(math.Round(trailWidthAt(distances[i]/total, headWidth)))
		if radius < 1 {
			radius = 1
		}
		diameter := radius * 2
		_ = win32.GDIPlusFillEllipse(hdc, points[i].X-radius, points[i].Y-radius, diameter, diameter, argb)
	}

	head := points[len(points)-1]
	radius := int32(math.Round(headWidth / 2))
	if radius < 2 {
		radius = 2
	}
	diameter := radius * 2
	_ = win32.GDIPlusFillEllipse(hdc, head.X-radius, head.Y-radius, diameter, diameter, argb)
}

func trailSegmentPolygon(from, to win32.POINT, fromWidth, toWidth float64) []win32.POINT {
	nx, ny := segmentNormal(from, to)
	return []win32.POINT{
		{
			X: int32(math.Round(float64(from.X) + (nx * fromWidth))),
			Y: int32(math.Round(float64(from.Y) + (ny * fromWidth))),
		},
		{
			X: int32(math.Round(float64(to.X) + (nx * toWidth))),
			Y: int32(math.Round(float64(to.Y) + (ny * toWidth))),
		},
		{
			X: int32(math.Round(float64(to.X) - (nx * toWidth))),
			Y: int32(math.Round(float64(to.Y) - (ny * toWidth))),
		},
		{
			X: int32(math.Round(float64(from.X) - (nx * fromWidth))),
			Y: int32(math.Round(float64(from.Y) - (ny * fromWidth))),
		},
	}
}

func cumulativeDistances(points []win32.POINT) []float64 {
	distances := make([]float64, len(points))
	for i := 1; i < len(points); i++ {
		distances[i] = distances[i-1] + pointDistance(points[i-1], points[i])
	}
	return distances
}

func trailWidthAt(progress, headWidth float64) float64 {
	return (trailTipWidth + ((headWidth - trailTipWidth) * progress)) / 2
}

func segmentNormal(from, to win32.POINT) (float64, float64) {
	dx := float64(to.X - from.X)
	dy := float64(to.Y - from.Y)
	length := math.Hypot(dx, dy)
	if length == 0 {
		return 0, -1
	}
	return -dy / length, dx / length
}

func trailBounds(points []win32.POINT) (int32, int32, int32, int32) {
	left := points[0].X
	top := points[0].Y
	right := points[0].X
	bottom := points[0].Y
	for _, point := range points[1:] {
		if point.X < left {
			left = point.X
		}
		if point.Y < top {
			top = point.Y
		}
		if point.X > right {
			right = point.X
		}
		if point.Y > bottom {
			bottom = point.Y
		}
	}
	return left, top, right, bottom
}

func interpolateTrailSample(from, to trailSample, progress float64) trailSample {
	if progress < 0 {
		progress = 0
	}
	if progress > 1 {
		progress = 1
	}
	return trailSample{
		Point: win32.POINT{
			X: int32(math.Round(float64(from.Point.X) + (float64(to.Point.X-from.Point.X) * progress))),
			Y: int32(math.Round(float64(from.Point.Y) + (float64(to.Point.Y-from.Point.Y) * progress))),
		},
		At: from.At.Add(time.Duration(float64(to.At.Sub(from.At)) * progress)),
	}
}

func (m *Manager) sustainUntil() time.Time {
	shakeDone := m.LastShakeAt.Add(sustainGrace)
	trailDone := m.lastTrailAt().Add(m.Settings.Fade)
	if trailDone.After(shakeDone) {
		return trailDone
	}
	return shakeDone
}

func (m *Manager) lastTrailAt() time.Time {
	if len(m.Trail) == 0 {
		return m.LastShakeAt
	}
	return m.Trail[len(m.Trail)-1].At
}

func pointDistance(a, b win32.POINT) float64 {
	return math.Hypot(float64(a.X-b.X), float64(a.Y-b.Y))
}

func colorRefToARGB(color win32.COLORREF) uint32 {
	raw := uint32(color)
	r := raw & 0xff
	g := (raw >> 8) & 0xff
	b := (raw >> 16) & 0xff
	return 0xff000000 | (r << 16) | (g << 8) | b
}

func normalizeTrailPixels(bits unsafe.Pointer, width, height int32, color win32.COLORREF) {
	if bits == nil || width <= 0 || height <= 0 {
		return
	}

	rawColor := uint32(color)
	red := rawColor & 0xff
	green := (rawColor >> 8) & 0xff
	blue := (rawColor >> 16) & 0xff
	pixels := unsafe.Slice((*uint32)(bits), int(width)*int(height))
	for i, pixel := range pixels {
		alpha := (pixel >> 24) & 0xff
		if alpha == 0 {
			continue
		}

		// UpdateLayeredWindow expects premultiplied BGRA. Keeping only the
		// rendered alpha prevents anti-aliased overlaps from creating pale seams.
		r := (red * alpha) / 255
		g := (green * alpha) / 255
		b := (blue * alpha) / 255
		pixels[i] = (alpha << 24) | (r << 16) | (g << 8) | b
	}
}

func normalizeSettings(settings Settings) Settings {
	if settings.Color == 0 {
		settings.Color = win32.RGB(80, 255, 210)
	}
	if settings.HeadWidth <= 0 {
		settings.HeadWidth = 13
	}
	if settings.MaxDistance <= 0 {
		settings.MaxDistance = 840
	}
	if settings.Fade <= 0 {
		settings.Fade = 260 * time.Millisecond
	}
	return settings
}
