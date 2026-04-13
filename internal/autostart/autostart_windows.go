//go:build windows

package autostart

import (
	"os"
	"path/filepath"

	"golang.org/x/sys/windows/registry"
)

const (
	RunKey = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`
	Name   = "OJTools"
)

func Command(exePath string) string {
	return `"` + exePath + `" run`
}

func CurrentValue() (string, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	value, _, err := key.GetStringValue(Name)
	if err == registry.ErrNotExist {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	return value, nil
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
	key, _, err := registry.CreateKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err != nil {
		return err
	}
	defer key.Close()

	return key.SetStringValue(Name, Command(cleanPath))
}

func Disable() error {
	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Microsoft\Windows\CurrentVersion\Run`, registry.SET_VALUE)
	if err == registry.ErrNotExist {
		return nil
	}
	if err != nil {
		return err
	}
	defer key.Close()

	if err := key.DeleteValue(Name); err == registry.ErrNotExist {
		return nil
	} else {
		return err
	}
}
