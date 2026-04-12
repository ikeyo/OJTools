//go:build windows

package autostart

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	RunKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	Name   = "OJTools"
)

func Command(exePath string) string {
	return `"` + exePath + `" run`
}

func CurrentValue() (string, error) {
	cmd := exec.Command("reg", "query", RunKey, "/v", Name)
	out, err := cmd.CombinedOutput()
	if err != nil {
		// `reg query` exits non-zero when the value is missing.
		if len(out) == 0 || strings.Contains(strings.ToLower(string(out)), "unable to find") {
			return "", nil
		}
		return "", err
	}

	text := string(out)
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, Name) {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		return strings.Join(fields[2:], " "), nil
	}

	return "", nil
}

func IsEnabled() (bool, error) {
	value, err := CurrentValue()
	if err != nil {
		return false, err
	}
	return value != "", nil
}

func EnableCurrentExecutable() error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}
	return Enable(exePath)
}

func Enable(exePath string) error {
	cleanPath := filepath.Clean(exePath)
	cmd := exec.Command(
		"reg",
		"add",
		RunKey,
		"/v", Name,
		"/t", "REG_SZ",
		"/d", Command(cleanPath),
		"/f",
	)
	return cmd.Run()
}

func Disable() error {
	cmd := exec.Command("reg", "delete", RunKey, "/v", Name, "/f")
	out, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.ToLower(string(out))
		if strings.Contains(text, "unable to find") {
			return nil
		}
		return err
	}
	return nil
}
