# OJTools Project Summary

## One-Line Summary

OJTools is a Windows tray utility that fixes cursor jump mismatches across dual monitors with different DPI scaling by letting the user calibrate the crossing and then applying that correction continuously.

## Problem

When two monitors have different scaling, resolution, or vertical placement, the pointer often crosses the shared edge at the wrong height. OJTools treats the primary monitor as the reference display and calibrates the secondary monitor against it so side-to-side mouse movement feels consistent.

## Solution

OJTools detects a side-by-side monitor pair, opens a visual calibration ruler on the boundary, and lets the user tune two values:

- `scale`: how much the secondary-side movement should stretch or compress
- `offset`: how much the mapped cursor should shift vertically

The overlay and remap logic both use the actual monitor bounds reported by Windows, including which monitor is primary and where the secondary monitor starts relative to the primary ruler. After saving, the runtime hook watches cursor motion and remaps the `Y` coordinate whenever the pointer crosses between the two displays.

## Current Feature Set

- Automatic side-by-side monitor pair detection
- Primary-monitor-based ruler and secondary-start calculation from real monitor coordinates
- Live calibration overlay with immediate crossing preview
- Saved per-pair cursor transform
- Tray icon runtime service
- Pause and resume from the tray
- Reset DPI calibration from the tray
- Keyboard lock toggle with `Ctrl+Alt+Shift+K`
- Select and block one specific key from the tray
- Windows auto-start toggle
- Save and restore desktop window layout
- Shake-to-find-cursor highlight with configurable delay
- Fullscreen-aware shake highlight suppression
- Windows GUI build with embedded app icon

## Intended Usage

This project is aimed at Windows users who work across two monitors and want cursor movement to feel physically aligned even when monitor scaling differs.

## Current Limitations

- Windows only
- Assumes the primary working pair is arranged horizontally
- Focused on one selected pair rather than a complex multi-monitor matrix
- Configuration is file-based and local to the executable or working directory
- Keyboard blocking is based on Windows virtual-key codes, so it is intentionally broad and not app-specific
- Window layout restore uses saved virtual desktop coordinates, so it works across multiple monitors when the monitor arrangement remains compatible, but it does not yet re-home windows based on changed monitor identity or a drastically different layout
- There is no in-app Git commit or push workflow; publishing source changes still happens through an external Git workflow

## Repository Purpose

This repository packages the source, build scripts, and project documentation needed to build OJTools locally, understand how it works, and extend the calibration/remap workflow over time.
