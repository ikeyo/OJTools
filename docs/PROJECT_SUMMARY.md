# OJTools Project Summary

## One-Line Summary

OJTools is a Windows tray utility that fixes cursor jump mismatches across dual monitors with different DPI scaling by letting the user calibrate the crossing and then applying that correction continuously.

## Problem

When two monitors have different scaling, resolution, or vertical placement, the pointer often crosses the shared edge at the wrong height. That makes side-to-side mouse movement feel inconsistent and interrupts normal desktop use.

## Solution

OJTools detects a side-by-side monitor pair, opens a visual calibration ruler on the boundary, and lets the user tune two values:

- `scale`: how much the right-side movement should stretch or compress
- `offset`: how much the mapped cursor should shift vertically

After saving, the runtime hook watches cursor motion and remaps the `Y` coordinate whenever the pointer crosses between the two displays.

## Current Feature Set

- Automatic side-by-side monitor pair detection
- Live calibration overlay with immediate crossing preview
- Saved per-pair cursor transform
- Tray icon runtime service
- Pause and resume from the tray
- Windows auto-start toggle
- Save and restore desktop window layout
- Shake-to-find-cursor highlight with configurable delay
- Fullscreen-aware shake highlight suppression

## Intended Usage

This project is aimed at Windows users who work across two monitors and want cursor movement to feel physically aligned even when monitor scaling differs.

## Current Limitations

- Windows only
- Assumes the primary working pair is arranged horizontally
- Focused on one selected pair rather than a complex multi-monitor matrix
- Configuration is file-based and local to the executable or working directory

## Repository Purpose

This repository packages the source, build scripts, and project documentation needed to build OJTools locally, understand how it works, and extend the calibration/remap workflow over time.
