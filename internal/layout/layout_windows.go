//go:build windows

package layout

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"ojtools/internal/win32"
)

type Placement struct {
	Flags          uint32      `json:"flags"`
	ShowCmd        uint32      `json:"showCmd"`
	MinPosition    win32.POINT `json:"minPosition"`
	MaxPosition    win32.POINT `json:"maxPosition"`
	NormalPosition win32.RECT  `json:"normalPosition"`
}

type WindowRecord struct {
	Title     string    `json:"title"`
	ClassName string    `json:"className"`
	ExePath   string    `json:"exePath"`
	Placement Placement `json:"placement"`
}

type Snapshot struct {
	Version         int            `json:"version"`
	SavedAt         time.Time      `json:"savedAt"`
	LastRestoredAt  *time.Time     `json:"lastRestoredAt,omitempty"`
	RestoreOnLaunch bool           `json:"restoreOnLaunch"`
	Windows         []WindowRecord `json:"windows"`
}

type RestoreResult struct {
	StoredWindows int
	Relaunched    int
	Repositioned  int
}

type SnapshotStatus struct {
	Exists          bool
	RestoreOnLaunch bool
	WindowCount     int
}

func DefaultPath() (string, error) {
	exePath, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exePath)
		lowerExe := strings.ToLower(exePath)
		if !strings.Contains(lowerExe, string(os.PathSeparator)+"go-build") {
			return filepath.Join(exeDir, "window-layout.json"), nil
		}
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(cwd, "window-layout.json"), nil
}

func SaveSnapshot(path string) (Snapshot, error) {
	windows, err := captureWindows()
	if err != nil {
		return Snapshot{}, err
	}
	if len(windows) == 0 {
		return Snapshot{}, errors.New("no restorable windows found")
	}

	snapshot := Snapshot{
		Version:         1,
		SavedAt:         time.Now(),
		RestoreOnLaunch: true,
		Windows:         windows,
	}
	if err := save(path, snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func Load(path string) (Snapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Snapshot{}, err
	}

	var snapshot Snapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return Snapshot{}, err
	}
	return snapshot, nil
}

func Status(path string) (SnapshotStatus, error) {
	snapshot, err := Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return SnapshotStatus{}, nil
		}
		return SnapshotStatus{}, err
	}

	return SnapshotStatus{
		Exists:          true,
		RestoreOnLaunch: snapshot.RestoreOnLaunch,
		WindowCount:     len(snapshot.Windows),
	}, nil
}

func SetRestoreOnLaunch(path string, enabled bool) (SnapshotStatus, error) {
	snapshot, err := Load(path)
	if err != nil {
		return SnapshotStatus{}, err
	}

	snapshot.RestoreOnLaunch = enabled
	if err := save(path, snapshot); err != nil {
		return SnapshotStatus{}, err
	}

	return SnapshotStatus{
		Exists:          true,
		RestoreOnLaunch: snapshot.RestoreOnLaunch,
		WindowCount:     len(snapshot.Windows),
	}, nil
}

func Clear(path string) error {
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}

func RestoreSnapshot(path string, clearPending bool) (RestoreResult, error) {
	snapshot, err := Load(path)
	if err != nil {
		return RestoreResult{}, err
	}
	if len(snapshot.Windows) == 0 {
		return RestoreResult{}, errors.New("saved window layout is empty")
	}

	result := RestoreResult{StoredWindows: len(snapshot.Windows)}
	used := map[win32.HWND]bool{}
	for _, record := range snapshot.Windows {
		hwnd, found := findWindow(record, used)
		if !found {
			if err := launch(record.ExePath); err == nil {
				result.Relaunched++
				hwnd, found = waitForWindow(record, used, 12*time.Second)
			}
		}

		if !found {
			continue
		}

		if applyPlacement(hwnd, record.Placement) == nil {
			result.Repositioned++
			used[hwnd] = true
		}
	}

	if clearPending {
		now := time.Now()
		snapshot.RestoreOnLaunch = false
		snapshot.LastRestoredAt = &now
		if err := save(path, snapshot); err != nil {
			return result, err
		}
	}

	return result, nil
}

func MaybeRestoreOnLaunch(path string) (bool, RestoreResult, error) {
	snapshot, err := Load(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return false, RestoreResult{}, nil
		}
		return false, RestoreResult{}, err
	}
	if !snapshot.RestoreOnLaunch || len(snapshot.Windows) == 0 {
		return false, RestoreResult{}, nil
	}

	result, err := RestoreSnapshot(path, true)
	return true, result, err
}

func captureWindows() ([]WindowRecord, error) {
	selfPID := uint32(os.Getpid())
	selfExe, _ := os.Executable()
	selfExe = strings.ToLower(filepath.Clean(selfExe))

	windows := make([]WindowRecord, 0, 16)
	err := win32.EnumWindows(func(hwnd win32.HWND) bool {
		if !win32.IsWindowVisible(hwnd) {
			return true
		}

		title, _ := win32.GetWindowText(hwnd)
		title = strings.TrimSpace(title)
		if title == "" {
			return true
		}

		className, _ := win32.GetClassName(hwnd)
		if skipClass(className) {
			return true
		}

		pid := win32.GetWindowThreadProcessId(hwnd)
		if pid == 0 || pid == selfPID {
			return true
		}

		exePath, err := processPath(pid)
		if err != nil || exePath == "" {
			return true
		}

		exeClean := strings.ToLower(filepath.Clean(exePath))
		if exeClean == selfExe || skipProcess(exeClean) {
			return true
		}

		var placement win32.WINDOWPLACEMENT
		if err := win32.GetWindowPlacement(hwnd, &placement); err != nil {
			return true
		}
		if isMinimizedShowCmd(placement.ShowCmd) {
			return true
		}
		savedBounds := placement.RcNormalPosition
		if win32.IsWindowArranged(hwnd) {
			if bounds, err := currentWindowBounds(hwnd); err == nil {
				savedBounds = bounds
			}
		}

		windows = append(windows, WindowRecord{
			Title:     title,
			ClassName: className,
			ExePath:   exePath,
			Placement: Placement{
				Flags:          placement.Flags,
				ShowCmd:        placement.ShowCmd,
				MinPosition:    placement.PtMinPosition,
				MaxPosition:    placement.PtMaxPosition,
				NormalPosition: savedBounds,
			},
		})

		return true
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(windows, func(i, j int) bool {
		if windows[i].ExePath == windows[j].ExePath {
			if windows[i].Title == windows[j].Title {
				return windows[i].ClassName < windows[j].ClassName
			}
			return windows[i].Title < windows[j].Title
		}
		return windows[i].ExePath < windows[j].ExePath
	})

	return windows, nil
}

func save(path string, snapshot Snapshot) error {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

func processPath(pid uint32) (string, error) {
	process, err := win32.OpenProcess(win32.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return "", err
	}
	defer win32.CloseHandle(process)

	return win32.QueryFullProcessImageName(process)
}

func currentWindowBounds(hwnd win32.HWND) (win32.RECT, error) {
	var rect win32.RECT
	if err := win32.DwmGetExtendedFrameBounds(hwnd, &rect); err == nil {
		return rect, nil
	}
	if err := win32.GetWindowRect(hwnd, &rect); err != nil {
		return win32.RECT{}, err
	}
	return rect, nil
}

func skipClass(className string) bool {
	switch strings.ToLower(className) {
	case "shell_traywnd", "progman", "workerw":
		return true
	default:
		return false
	}
}

func skipProcess(exePath string) bool {
	switch strings.ToLower(filepath.Base(exePath)) {
	case "ojtools.exe", "applicationframehost.exe", "explorer.exe", "searchhost.exe":
		return true
	default:
		return false
	}
}

func launch(exePath string) error {
	cmd := exec.Command(exePath)
	cmd.Dir = filepath.Dir(exePath)
	return cmd.Start()
}

func waitForWindow(record WindowRecord, used map[win32.HWND]bool, timeout time.Duration) (win32.HWND, bool) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if hwnd, ok := findWindow(record, used); ok {
			return hwnd, true
		}
		time.Sleep(300 * time.Millisecond)
	}
	return 0, false
}

func findWindow(record WindowRecord, used map[win32.HWND]bool) (win32.HWND, bool) {
	var best win32.HWND
	bestScore := -1

	_ = win32.EnumWindows(func(hwnd win32.HWND) bool {
		if used[hwnd] {
			return true
		}

		pid := win32.GetWindowThreadProcessId(hwnd)
		if pid == 0 {
			return true
		}

		exePath, err := processPath(pid)
		if err != nil || !strings.EqualFold(filepath.Clean(exePath), filepath.Clean(record.ExePath)) {
			return true
		}

		title, _ := win32.GetWindowText(hwnd)
		className, _ := win32.GetClassName(hwnd)
		score := 1
		if win32.IsWindowVisible(hwnd) {
			score += 4
		}
		if strings.TrimSpace(title) != "" {
			score++
		}
		if strings.EqualFold(strings.TrimSpace(title), strings.TrimSpace(record.Title)) {
			score += 4
		}
		if strings.EqualFold(className, record.ClassName) {
			score += 2
		}

		if score > bestScore {
			best = hwnd
			bestScore = score
		}
		return true
	})

	return best, best != 0
}

func applyPlacement(hwnd win32.HWND, placement Placement) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := applyPlacementOnce(hwnd, placement); err == nil {
			return nil
		} else {
			lastErr = err
		}

		time.Sleep(250 * time.Millisecond)
	}

	return lastErr
}

func applyPlacementOnce(hwnd win32.HWND, placement Placement) error {
	windowPlacement := win32.WINDOWPLACEMENT{
		Flags:            placement.Flags,
		ShowCmd:          placement.ShowCmd,
		PtMinPosition:    placement.MinPosition,
		PtMaxPosition:    placement.MaxPosition,
		RcNormalPosition: placement.NormalPosition,
	}
	if err := win32.SetWindowPlacement(hwnd, &windowPlacement); err != nil {
		return err
	}

	width := placement.NormalPosition.Width()
	height := placement.NormalPosition.Height()
	if width > 0 && height > 0 {
		win32.ShowWindow(hwnd, win32.SW_RESTORE)
		if err := win32.SetWindowPos(
			hwnd,
			0,
			placement.NormalPosition.Left,
			placement.NormalPosition.Top,
			width,
			height,
			win32.SWP_NOZORDER|win32.SWP_NOACTIVATE,
		); err != nil {
			return err
		}
	}

	switch placement.ShowCmd {
	case win32.SW_SHOWMAXIMIZED:
		win32.ShowWindow(hwnd, win32.SW_SHOWMAXIMIZED)
	case win32.SW_SHOWMINIMIZED, win32.SW_MINIMIZE:
		win32.ShowWindow(hwnd, int32(placement.ShowCmd))
	default:
		win32.ShowWindow(hwnd, win32.SW_SHOWNORMAL)
	}

	if !isMinimizedShowCmd(placement.ShowCmd) && placement.ShowCmd != win32.SW_SHOWMAXIMIZED {
		time.Sleep(150 * time.Millisecond)

		var rect win32.RECT
		if err := win32.GetWindowRect(hwnd, &rect); err != nil {
			return err
		}
		if !rectCloseEnough(rect, placement.NormalPosition, 24) {
			return errors.New("window did not keep restored position")
		}
	}

	return nil
}

func isMinimizedShowCmd(showCmd uint32) bool {
	return showCmd == win32.SW_SHOWMINIMIZED || showCmd == win32.SW_MINIMIZE
}

func rectCloseEnough(actual, expected win32.RECT, tolerance int32) bool {
	return abs32(actual.Left-expected.Left) <= tolerance &&
		abs32(actual.Top-expected.Top) <= tolerance &&
		abs32(actual.Right-expected.Right) <= tolerance &&
		abs32(actual.Bottom-expected.Bottom) <= tolerance
}

func abs32(v int32) int32 {
	if v < 0 {
		return -v
	}
	return v
}
