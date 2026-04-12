package calibration

import (
	"ojtools/internal/config"
	"ojtools/internal/monitor"
)

func DefaultTransform(pair monitor.Pair) config.Transform {
	if pair.Left.DpiY == 0 || pair.Right.DpiY == 0 {
		return config.Transform{Scale: 1, Offset: 0}
	}

	return config.Transform{
		Scale:  float64(pair.Right.DpiY) / float64(pair.Left.DpiY),
		Offset: 0,
	}
}

func MapLeftToRight(pair monitor.Pair, transform config.Transform, srcY float64) float64 {
	return float64(pair.Right.Bounds.Top) +
		((srcY - float64(pair.Left.Bounds.Top)) * transform.Scale) +
		transform.Offset
}

func MapRightToLeft(pair monitor.Pair, transform config.Transform, srcY float64) float64 {
	if transform.Scale == 0 {
		return srcY
	}

	return float64(pair.Left.Bounds.Top) +
		((srcY - float64(pair.Right.Bounds.Top) - transform.Offset) / transform.Scale)
}
