package monitor

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"syscall"

	"ojtools/internal/win32"
)

type Monitor struct {
	Handle     win32.HMONITOR
	DeviceName string
	Bounds     win32.RECT
	WorkArea   win32.RECT
	Primary    bool
	DpiX       uint32
	DpiY       uint32
	ScaleX     float64
	ScaleY     float64
}

type Pair struct {
	Left          Monitor
	Right         Monitor
	BoundaryX     int32
	OverlapTop    int32
	OverlapBottom int32
}

func (p Pair) PrimaryMonitor() Monitor {
	if p.Left.Primary && !p.Right.Primary {
		return p.Left
	}
	if p.Right.Primary && !p.Left.Primary {
		return p.Right
	}
	return p.Left
}

func (p Pair) SecondaryMonitor() Monitor {
	if p.Left.Primary && !p.Right.Primary {
		return p.Right
	}
	if p.Right.Primary && !p.Left.Primary {
		return p.Left
	}
	return p.Right
}

func (p Pair) PrimaryOnLeft() bool {
	primary := p.PrimaryMonitor()
	return primary.DeviceName == p.Left.DeviceName
}

func (p Pair) ContainsOverlapY(y float64) bool {
	return y >= float64(p.OverlapTop) && y < float64(p.OverlapBottom)
}

func (p Pair) CrossingFromLeftToRight(prev, current win32.POINT) (float64, bool) {
	if !pointInside(prev, p.Left.Bounds) || current.X < p.Right.Bounds.Left {
		return 0, false
	}

	y, ok := crossingYAtX(prev, current, p.Right.Bounds.Left)
	if !ok || !p.ContainsOverlapY(y) {
		return 0, false
	}

	return y, true
}

func (p Pair) CrossingFromRightToLeft(prev, current win32.POINT) (float64, bool) {
	if !pointInside(prev, p.Right.Bounds) || current.X >= p.Left.Bounds.Right {
		return 0, false
	}

	y, ok := crossingYAtX(prev, current, p.Left.Bounds.Right)
	if !ok || !p.ContainsOverlapY(y) {
		return 0, false
	}

	return y, true
}

func Enumerate() ([]Monitor, error) {
	monitors := make([]Monitor, 0, 4)

	err := win32.EnumDisplayMonitors(func(handle win32.HMONITOR, _ win32.HDC, _ *win32.RECT) bool {
		var info win32.MONITORINFOEX
		if err := win32.GetMonitorInfo(handle, &info); err != nil {
			return true
		}

		dpiX, dpiY, err := win32.GetDpiForMonitor(handle)
		if err != nil {
			dpiX = 96
			dpiY = 96
		}

		monitors = append(monitors, Monitor{
			Handle:     handle,
			DeviceName: syscall.UTF16ToString(info.SzDevice[:]),
			Bounds:     info.RcMonitor,
			WorkArea:   info.RcWork,
			Primary:    info.DwFlags&win32.MONITORINFOF_PRIMARY != 0,
			DpiX:       dpiX,
			DpiY:       dpiY,
			ScaleX:     float64(dpiX) / 96.0,
			ScaleY:     float64(dpiY) / 96.0,
		})

		return true
	})
	if err != nil {
		return nil, err
	}
	if len(monitors) == 0 {
		return nil, errors.New("no monitors detected")
	}

	sort.Slice(monitors, func(i, j int) bool {
		if monitors[i].Bounds.Left == monitors[j].Bounds.Left {
			return monitors[i].Bounds.Top < monitors[j].Bounds.Top
		}
		return monitors[i].Bounds.Left < monitors[j].Bounds.Left
	})

	return monitors, nil
}

func PickHorizontalPair(monitors []Monitor) (Pair, error) {
	if len(monitors) < 2 {
		return Pair{}, errors.New("at least two monitors are required")
	}

	bestGap := int32(math.MaxInt32)
	bestOverlap := int32(-1)
	var best Pair
	found := false

	for i := 0; i < len(monitors); i++ {
		for j := i + 1; j < len(monitors); j++ {
			left, right := ordered(monitors[i], monitors[j])
			overlapTop := max32(left.Bounds.Top, right.Bounds.Top)
			overlapBottom := min32(left.Bounds.Bottom, right.Bounds.Bottom)
			overlap := overlapBottom - overlapTop
			if overlap <= 0 {
				continue
			}

			gap := abs32(right.Bounds.Left - left.Bounds.Right)
			if gap < bestGap || (gap == bestGap && overlap > bestOverlap) {
				bestGap = gap
				bestOverlap = overlap
				best = Pair{
					Left:          left,
					Right:         right,
					BoundaryX:     left.Bounds.Right + gap/2,
					OverlapTop:    overlapTop,
					OverlapBottom: overlapBottom,
				}
				found = true
			}
		}
	}

	if !found {
		return Pair{}, errors.New("could not find a side-by-side monitor pair")
	}

	return best, nil
}

func PairKey(pair Pair) string {
	return fmt.Sprintf("%s|%s", pair.Left.DeviceName, pair.Right.DeviceName)
}

func VirtualBounds(monitors []Monitor) win32.RECT {
	if len(monitors) == 0 {
		return win32.RECT{}
	}

	out := monitors[0].Bounds
	for _, m := range monitors[1:] {
		out.Left = min32(out.Left, m.Bounds.Left)
		out.Top = min32(out.Top, m.Bounds.Top)
		out.Right = max32(out.Right, m.Bounds.Right)
		out.Bottom = max32(out.Bottom, m.Bounds.Bottom)
	}
	return out
}

func ordered(a, b Monitor) (Monitor, Monitor) {
	if a.Bounds.Left < b.Bounds.Left {
		return a, b
	}
	return b, a
}

func pointInside(point win32.POINT, rect win32.RECT) bool {
	return point.X >= rect.Left &&
		point.X < rect.Right &&
		point.Y >= rect.Top &&
		point.Y < rect.Bottom
}

func crossingYAtX(prev, current win32.POINT, x int32) (float64, bool) {
	dx := current.X - prev.X
	if dx == 0 {
		return 0, false
	}

	t := float64(x-prev.X) / float64(dx)
	if t < 0 || t > 1 {
		return 0, false
	}

	return float64(prev.Y) + (float64(current.Y-prev.Y) * t), true
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}

func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}

func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
