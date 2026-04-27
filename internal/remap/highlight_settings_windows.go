//go:build windows

package remap

import (
	"time"

	"ojtools/internal/config"
	"ojtools/internal/highlight"
	"ojtools/internal/win32"
)

const (
	trailColorMint   = "mint"
	trailColorYellow = "yellow"
	trailColorPink   = "pink"
	trailColorCyan   = "cyan"
)

func highlightSettingsFromFeatures(features config.Features) highlight.Settings {
	return highlight.Settings{
		Color:       trailColor(features.ShakeTrailColor),
		HeadWidth:   float64(normalizeTrailThickness(features.ShakeTrailThickness)),
		MaxDistance: float64(normalizeTrailLength(features.ShakeTrailLength)),
		Fade:        time.Duration(normalizeTrailFadeMs(features.ShakeTrailFadeMs)) * time.Millisecond,
	}
}

func (s *Service) setTrailColor(color string) {
	s.Config.Features.ShakeTrailColor = normalizeTrailColor(color)
	s.saveAndApplyHighlightSettings()
}

func (s *Service) setTrailThickness(thickness int) {
	s.Config.Features.ShakeTrailThickness = normalizeTrailThickness(thickness)
	s.saveAndApplyHighlightSettings()
}

func (s *Service) setTrailLength(length int) {
	s.Config.Features.ShakeTrailLength = normalizeTrailLength(length)
	s.saveAndApplyHighlightSettings()
}

func (s *Service) setTrailFadeMs(fadeMs int) {
	s.Config.Features.ShakeTrailFadeMs = normalizeTrailFadeMs(fadeMs)
	s.saveAndApplyHighlightSettings()
}

func (s *Service) saveAndApplyHighlightSettings() {
	s.Config.Features.ShakeTrailColor = normalizeTrailColor(s.Config.Features.ShakeTrailColor)
	s.Config.Features.ShakeTrailThickness = normalizeTrailThickness(s.Config.Features.ShakeTrailThickness)
	s.Config.Features.ShakeTrailLength = normalizeTrailLength(s.Config.Features.ShakeTrailLength)
	s.Config.Features.ShakeTrailFadeMs = normalizeTrailFadeMs(s.Config.Features.ShakeTrailFadeMs)
	_ = saveAppConfig(s.ConfigPath, s.Config)
	if s.Highlight != nil {
		s.Highlight.SetSettings(highlightSettingsFromFeatures(s.Config.Features))
	}
}

func normalizeTrailColor(color string) string {
	switch color {
	case trailColorYellow, trailColorPink, trailColorCyan:
		return color
	default:
		return trailColorMint
	}
}

func normalizeTrailThickness(thickness int) int {
	switch thickness {
	case 8, 13, 20:
		return thickness
	default:
		return 13
	}
}

func normalizeTrailLength(length int) int {
	switch length {
	case 280, 840, 1680:
		return length
	default:
		return 840
	}
}

func normalizeTrailFadeMs(fadeMs int) int {
	switch fadeMs {
	case 160, 260, 520:
		return fadeMs
	default:
		return 260
	}
}

func trailColor(color string) win32.COLORREF {
	switch normalizeTrailColor(color) {
	case trailColorYellow:
		return win32.RGB(245, 255, 80)
	case trailColorPink:
		return win32.RGB(255, 80, 220)
	case trailColorCyan:
		return win32.RGB(80, 190, 255)
	default:
		return win32.RGB(80, 255, 210)
	}
}
