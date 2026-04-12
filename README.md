# OJTools

`OJTools` is a Windows-only Go utility for calibrating and remapping cursor movement across a dual-monitor boundary when each display uses a different DPI scaling or vertical alignment.

The project focuses on a common annoyance: the mouse pointer lands too high or too low after crossing from one monitor to the other. OJTools lets you calibrate that crossing visually, then keeps the correction running from the tray.

## What It Does

- Detects the best side-by-side monitor pair automatically
- Opens a calibration overlay with live preview while you test cursor crossing
- Saves a reusable `scale` and `offset` transform per monitor pair
- Runs a tray-based background remapper for everyday use
- Supports Windows auto-start from the current user registry
- Can save and restore the current desktop window layout
- Includes a shake-to-find-cursor highlight with configurable delay

## How Calibration Works

The left monitor is treated as the reference ruler.

- `scale` stretches or compresses the right-side mapping
- `offset` shifts the mapped cursor position up or down
- The saved transform is reused whenever the cursor crosses that monitor pair again

## Commands

Run from source:

```powershell
go run .\cmd\ojtools inspect
go run .\cmd\ojtools calibrate
go run .\cmd\ojtools run
```

Build a Windows GUI executable:

```powershell
go build -ldflags "-H=windowsgui" -o .\OJTools.exe .\cmd\ojtools
.\OJTools.exe calibrate
.\OJTools.exe run
```

Or use the bundled build helper:

```powershell
.\build.ps1
```

Manage Windows auto-start:

```powershell
.\autostart.ps1 enable
.\autostart.ps1 status
.\autostart.ps1 disable
```

## Calibration Controls

- Mouse wheel: adjust `scale`
- `Shift` + mouse wheel: fine scale adjustment
- Left mouse drag: adjust `offset`
- Arrow left/right: adjust `scale`
- Arrow up/down: adjust `offset`
- `Enter` or `S`: save to `config.json` and apply immediately
- `R`: reset to the automatic DPI-based guess
- `Esc`: close without saving

While the calibration overlay is open, test crossings use the current unsaved value immediately so you can tune by feel.

## Tray Features

When `run` is active, right-click the tray icon to access:

- `Cursor Calibration...`
- `Pause Cursor Remap`
- `Auto-Start`
- `Restore Layout On Launch`
- `Shake To Find Cursor`
- `Shake Delay: 0.5 s / 1 s / 2 s`
- `Save Window Layout`
- `Restore Window Layout`
- `Clear Saved Window Layout`
- `Open Config Folder`
- `About`
- `Exit OJTools`

Double-clicking the tray icon opens calibration directly.

## Configuration Files

- `config.json`: saved cursor calibration and feature settings
- `window-layout.json`: saved desktop window placement snapshot

Both files are written next to the built executable, or to the current working directory when using `go run`.

## Notes And Limits

- The current implementation assumes the main monitor pair is arranged side-by-side.
- Auto-start uses `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` and launches `OJTools.exe run`.
- Saving a window layout marks it for one automatic restore on the next launch, which is useful after sign-in or reboot.
- Tray checkmarks reflect paused remap state, auto-start state, restore-on-launch state, and shake highlight state.
- The shake highlight stays suppressed while a fullscreen foreground app is active.
- The first run can start from the Windows DPI ratio, then be refined manually with the ruler overlay.

## Repository Layout

- `cmd/ojtools`: application entry point
- `internal/overlay`: calibration overlay UI
- `internal/remap`: runtime cursor remap service and tray menu
- `internal/layout`: window layout snapshot and restore support
- `internal/highlight`: shake-to-find-cursor highlight overlay
- `internal/monitor`: monitor enumeration and pair selection
- `internal/config`: persisted configuration model

Additional project context is available in `docs/PROJECT_SUMMARY.md`.
