package calibration

import (
	"ojtools/internal/config"
	"ojtools/internal/monitor"
)

func DefaultTransform(pair monitor.Pair) config.Transform {
	primary := pair.PrimaryMonitor()
	secondary := pair.SecondaryMonitor()
	if primary.DpiY == 0 || secondary.DpiY == 0 {
		return config.Transform{Scale: 1, Offset: 0}
	}

	return config.Transform{
		Scale:  float64(secondary.DpiY) / float64(primary.DpiY),
		Offset: 0,
	}
}

func MapPrimaryToSecondary(pair monitor.Pair, transform config.Transform, srcY float64) float64 {
	secondary := pair.SecondaryMonitor()

	return float64(secondary.Bounds.Top) +
		((srcY - float64(secondary.Bounds.Top)) * transform.Scale) +
		transform.Offset
}

func MapSecondaryToPrimary(pair monitor.Pair, transform config.Transform, srcY float64) float64 {
	if transform.Scale == 0 {
		return srcY
	}

	secondary := pair.SecondaryMonitor()

	return float64(secondary.Bounds.Top) +
		((srcY - float64(secondary.Bounds.Top) - transform.Offset) / transform.Scale)
}
