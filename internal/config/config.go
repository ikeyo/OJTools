package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"
)

type Transform struct {
	Scale  float64 `json:"scale"`
	Offset float64 `json:"offset"`
}

type PairCalibration struct {
	PairKey     string    `json:"pairKey"`
	LeftDevice  string    `json:"leftDevice"`
	RightDevice string    `json:"rightDevice"`
	Transform   Transform `json:"transform"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Features struct {
	ShakeCursorHighlight  bool   `json:"shakeCursorHighlight"`
	ShakeHighlightDelayMs int    `json:"shakeHighlightDelayMs"`
	BlockedKeyEnabled     bool   `json:"blockedKeyEnabled"`
	BlockedVK             uint32 `json:"blockedVK"`
}

type Config struct {
	Version      int                        `json:"version"`
	Calibrations map[string]PairCalibration `json:"calibrations"`
	Features     Features                   `json:"features"`
}

func Default() Config {
	return Config{
		Version:      1,
		Calibrations: map[string]PairCalibration{},
		Features: Features{
			ShakeCursorHighlight:  true,
			ShakeHighlightDelayMs: 500,
			BlockedKeyEnabled:     false,
			BlockedVK:             0,
		},
	}
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return Default(), nil
		}
		return Config{}, err
	}

	cfg := Default()
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if cfg.Calibrations == nil {
		cfg.Calibrations = map[string]PairCalibration{}
	}

	return cfg, nil
}

func Save(path string, cfg Config) error {
	if cfg.Calibrations == nil {
		cfg.Calibrations = map[string]PairCalibration{}
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o644)
}

func (c *Config) Put(cal PairCalibration) {
	if c.Calibrations == nil {
		c.Calibrations = map[string]PairCalibration{}
	}
	c.Calibrations[cal.PairKey] = cal
}

func (c Config) Get(pairKey string) (PairCalibration, bool) {
	cal, ok := c.Calibrations[pairKey]
	return cal, ok
}

func (c *Config) Delete(pairKey string) {
	if c.Calibrations == nil {
		return
	}
	delete(c.Calibrations, pairKey)
}
