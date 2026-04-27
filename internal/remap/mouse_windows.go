//go:build windows

package remap

import (
	"math"
	"time"
	"unsafe"

	"ojtools/internal/calibration"
	"ojtools/internal/win32"
)

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
