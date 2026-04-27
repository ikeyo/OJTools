//go:build windows

package remap

import (
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"ojtools/internal/calibration"
	"ojtools/internal/config"
	"ojtools/internal/layout"
	"ojtools/internal/monitor"
)

func (s *Service) startCalibration() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	cmd := exec.Command(exePath, "calibrate")
	cmd.Dir = filepath.Dir(exePath)
	return cmd.Start()
}

func (s *Service) resetCalibration() error {
	pairKey := monitor.PairKey(s.Pair)
	s.Config.Delete(pairKey)
	s.Transform = calibration.DefaultTransform(s.Pair)
	s.HavePrev = false
	return saveAppConfig(s.ConfigPath, s.Config)
}

func normalizeShakeDelay(delayMs int) int {
	switch delayMs {
	case 500, 1000, 2000:
		return delayMs
	default:
		return 500
	}
}

func (s *Service) restoreSavedLayoutOnLaunch() {
	time.Sleep(4 * time.Second)
	_, _, _ = restoreWindowLayout(true)
}

func saveWindowLayout() error {
	path, err := layout.DefaultPath()
	if err != nil {
		return err
	}
	_, err = layout.SaveSnapshot(path)
	return err
}

func restoreWindowLayout(clearPending bool) (bool, layout.RestoreResult, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return false, layout.RestoreResult{}, err
	}

	if clearPending {
		return layout.MaybeRestoreOnLaunch(path)
	}

	result, err := layout.RestoreSnapshot(path, false)
	return true, result, err
}

func clearWindowLayout() error {
	path, err := layout.DefaultPath()
	if err != nil {
		return err
	}
	return layout.Clear(path)
}

func windowLayoutStatus() (layout.SnapshotStatus, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return layout.SnapshotStatus{}, err
	}
	return layout.Status(path)
}

func setRestoreOnLaunch(enabled bool) (layout.SnapshotStatus, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return layout.SnapshotStatus{}, err
	}
	return layout.SetRestoreOnLaunch(path, enabled)
}

func openConfigFolder() error {
	path, err := layout.DefaultPath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	cmd := exec.Command("explorer.exe", dir)
	cmd.Dir = dir
	return cmd.Start()
}

func appConfigDir() (string, error) {
	path, err := layout.DefaultPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(path), nil
}

func saveAppConfig(path string, cfg config.Config) error {
	if path == "" {
		return nil
	}
	return config.Save(path, cfg)
}
