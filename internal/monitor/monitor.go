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
