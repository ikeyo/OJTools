//go:build windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"ojtools/internal/calibration"
	"ojtools/internal/config"
	"ojtools/internal/monitor"
	"ojtools/internal/overlay"
	"ojtools/internal/remap"
	"ojtools/internal/win32"
)

func main() {
	if err := win32.SetPerMonitorV2(); err != nil {
		// Ignore if Windows has already set DPI awareness for this process.
	}

	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	command := "calibrate"
	if len(args) > 0 {
		command = args[0]
	}

	switch command {
	case "inspect":
		return runInspect()
	case "calibrate":
		return runCalibrate()
	case "run":
		return runRemapper()
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command %q", command)
	}
}

func runInspect() error {
	monitors, pair, err := detectPair()
	if err != nil {
		return err
	}

	fmt.Println("Detected monitors:")
	for _, m := range monitors {
		fmt.Printf(
			"- %s bounds=(%d,%d)-(%d,%d) resolution=%dx%d scale=%.0f%% dpi=%d primary=%v\n",
			m.DeviceName,
			m.Bounds.Left,
			m.Bounds.Top,
			m.Bounds.Right,
			m.Bounds.Bottom,
			m.Bounds.Width(),
			m.Bounds.Height(),
			m.ScaleY*100,
			m.DpiY,
			m.Primary,
		)
	}

	fmt.Println()
	fmt.Printf("Selected pair: %s <-> %s\n", pair.Left.DeviceName, pair.Right.DeviceName)
	fmt.Printf("Boundary X: %d  overlap=(%d..%d)\n", pair.BoundaryX, pair.OverlapTop, pair.OverlapBottom)

	return nil
}

func runCalibrate() error {
	_, pair, err := detectPair()
	if err != nil {
		return err
	}

	cfgPath, err := configPath()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	autoTransform := calibration.DefaultTransform(pair)
	transform := autoTransform
	if saved, ok := cfg.Get(monitor.PairKey(pair)); ok {
		transform = saved.Transform
	}

	fmt.Printf("Opening calibration overlay for %s <-> %s\n", pair.Left.DeviceName, pair.Right.DeviceName)
	fmt.Printf("Config path: %s\n", cfgPath)

	result, err := overlay.Run(pair, transform, autoTransform, func(t config.Transform) error {
		cfg.Put(config.PairCalibration{
			PairKey:     monitor.PairKey(pair),
			LeftDevice:  pair.Left.DeviceName,
			RightDevice: pair.Right.DeviceName,
			Transform:   t,
			UpdatedAt:   time.Now(),
		})
		return config.Save(cfgPath, cfg)
	})
	if err != nil {
		return err
	}
	if !result.Saved {
		fmt.Println("Calibration window closed without saving.")
		return nil
	}

	fmt.Printf("Saved calibration scale=%.4f offset=%.1f\n", result.Transform.Scale, result.Transform.Offset)
	fmt.Println("Applying immediately. Press Ctrl+C in this console to stop.")
	return remap.Run(pair, result.Transform, cfgPath, cfg)
}

func runRemapper() error {
	_, pair, err := detectPair()
	if err != nil {
		return err
	}

	cfgPath, err := configPath()
	if err != nil {
		return err
	}

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	transform := calibration.DefaultTransform(pair)
	if saved, ok := cfg.Get(monitor.PairKey(pair)); ok {
		transform = saved.Transform
		fmt.Printf("Using saved calibration scale=%.4f offset=%.1f\n", transform.Scale, transform.Offset)
	} else {
		fmt.Printf("No saved calibration found, using DPI guess scale=%.4f offset=%.1f\n", transform.Scale, transform.Offset)
	}

	fmt.Printf("Running remapper for %s <-> %s\n", pair.Left.DeviceName, pair.Right.DeviceName)
	fmt.Println("Press Ctrl+C in this console to stop.")

	return remap.Run(pair, transform, cfgPath, cfg)
}

func detectPair() ([]monitor.Monitor, monitor.Pair, error) {
	monitors, err := monitor.Enumerate()
	if err != nil {
		return nil, monitor.Pair{}, err
	}

	pair, err := monitor.PickHorizontalPair(monitors)
	if err != nil {
		return nil, monitor.Pair{}, err
	}

	return monitors, pair, nil
}

func configPath() (string, error) {
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		lowerExe := strings.ToLower(exePath)
		if !strings.Contains(lowerExe, string(os.PathSeparator)+"go-build") {
			return filepath.Join(exeDir, "config.json"), nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, "config.json"), nil
}

func printUsage() {
	fmt.Println("OJTools")
	fmt.Println()
	fmt.Println("Usage:")
	fmt.Println("  OJTools inspect")
	fmt.Println("  OJTools calibrate")
	fmt.Println("  OJTools run")
}
