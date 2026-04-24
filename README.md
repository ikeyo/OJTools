# OJTools

`OJTools` is a Windows-only Go utility for calibrating and remapping cursor movement across a dual-monitor boundary when each display uses a different DPI scaling or vertical alignment.

The project focuses on a common annoyance: the mouse pointer lands too high or too low after crossing from one monitor to the other. OJTools lets you calibrate that crossing visually, then keeps the correction running from the tray.

## What It Does

- Detects the best side-by-side monitor pair automatically
- Opens a calibration overlay with live preview while you test cursor crossing
- Uses the primary monitor as the reference ruler and derives the secondary monitor start position from the real Windows monitor layout
- Saves a reusable `scale` and `offset` transform per monitor pair
- Runs a tray-based background remapper for everyday use
- Supports Windows auto-start from the current user registry
- Can save and restore the current desktop window layout
- Includes a shake-to-find-cursor highlight with configurable delay
- Can lock the keyboard or block one selected key from the tray

## How Calibration Works

The primary monitor is treated as the reference ruler, and the secondary monitor is calibrated against it.

- `scale` stretches or compresses the secondary-side mapping
- `offset` shifts the mapped cursor position up or down
- The ruler overlay uses the actual monitor bounds reported by Windows, so the secondary side starts from the matching position on the primary ruler instead of resetting to zero
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

The build helper generates a Windows GUI executable, downloads a Material icon source, and embeds the application icon resource automatically.

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
- `Reset DPI Calibration`
- `Pause Cursor Remap`
- `Lock Keyboard (Ctrl+Alt+Shift+K)`
- `Blocked Key: ...`
- `Block Specific Key`
- `Set Blocked Key (Next Press)`
- `Clear Blocked Key`
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

- `config.json`: saved cursor calibration and feature settings, including the blocked-key setting
- `window-layout.json`: saved desktop window placement snapshot

Both files are written next to the built executable, or to the current working directory when using `go run`.

## Notes And Limits

- The current implementation assumes the active monitor pair is arranged side-by-side, but it also respects the real vertical offset and overlap reported by Windows for that pair.
- Auto-start uses `HKCU\Software\Microsoft\Windows\CurrentVersion\Run` and launches `OJTools.exe run`.
- Saving a window layout stores the Windows virtual desktop placement for each captured window, so multi-monitor layouts can be restored as long as the monitor arrangement stays compatible.
- If monitor count, primary monitor, scaling, or relative placement changes significantly after saving, some restored windows can land on a different monitor or off-screen because layout restore currently saves window placement coordinates, not per-monitor attachment metadata.
- Tray checkmarks reflect paused remap state, keyboard lock state, blocked-key state, auto-start state, restore-on-launch state, and shake highlight state.
- Keyboard lock blocks key input until toggled off with `Ctrl+Alt+Shift+K` or the tray menu. The blocked-key feature only suppresses the selected virtual key.
- The shake highlight stays suppressed while a fullscreen foreground app is active.
- The first run can start from the Windows DPI ratio, then be refined manually with the ruler overlay.
- OJTools itself does not include an in-app Git commit or push feature. Source changes are still published through a normal Git workflow outside the tray UI.

## Repository Layout

- `cmd/ojtools`: application entry point
- `internal/overlay`: calibration overlay UI
- `internal/remap`: runtime cursor remap service, tray menu, keyboard lock, blocked-key handling, and layout actions
- `internal/layout`: window layout snapshot and restore support
- `internal/highlight`: shake-to-find-cursor highlight overlay
- `internal/monitor`: monitor enumeration and pair selection
- `internal/config`: persisted configuration model

Additional project context is available in `docs/PROJECT_SUMMARY.md`.
