//go:build windows

package remap

import (
	"fmt"
	"unsafe"

	"ojtools/internal/win32"
)

func lowLevelKeyboardProc(nCode int32, wParam, lParam uintptr) uintptr {
	if activeService == nil {
		return 0
	}
	if nCode != win32.HC_ACTION {
		return win32.CallNextHookEx(activeService.KeyboardHook, nCode, wParam, lParam)
	}

	msg := uint32(wParam)
	keyboardInfo := (*win32.KBDLLHOOKSTRUCT)(unsafe.Pointer(lParam))
	keyDown := msg == win32.WM_KEYDOWN || msg == win32.WM_SYSKEYDOWN
	keyUp := msg == win32.WM_KEYUP || msg == win32.WM_SYSKEYUP

	if keyDown && activeService.CaptureNextBlockedKey {
		activeService.Config.Features.BlockedVK = keyboardInfo.VkCode
		activeService.Config.Features.BlockedKeyEnabled = true
		activeService.CaptureNextBlockedKey = false
		_ = saveAppConfig(activeService.ConfigPath, activeService.Config)
		activeService.refreshTrayTip()
		return 1
	}

	switch keyboardInfo.VkCode {
	case win32.VK_CONTROL, win32.VK_LCONTROL, win32.VK_RCONTROL:
		activeService.CtrlDown = keyDown
		if keyUp {
			activeService.CtrlDown = false
		}
	case win32.VK_MENU, win32.VK_LMENU, win32.VK_RMENU:
		activeService.AltDown = keyDown
		if keyUp {
			activeService.AltDown = false
		}
	case win32.VK_SHIFT, win32.VK_LSHIFT, win32.VK_RSHIFT:
		activeService.ShiftDown = keyDown
		if keyUp {
			activeService.ShiftDown = false
		}
	}

	if keyDown &&
		keyboardInfo.VkCode == win32.VK_K &&
		activeService.CtrlDown &&
		activeService.AltDown &&
		activeService.ShiftDown {
		activeService.toggleKeyboardLock()
		return 1
	}

	if activeService.KeyboardLocked {
		return 1
	}

	if activeService.Config.Features.BlockedKeyEnabled &&
		activeService.Config.Features.BlockedVK != 0 &&
		keyboardInfo.VkCode == activeService.Config.Features.BlockedVK {
		return 1
	}

	return win32.CallNextHookEx(activeService.KeyboardHook, nCode, wParam, lParam)
}

func (s *Service) toggleKeyboardLock() {
	s.KeyboardLocked = !s.KeyboardLocked
	if !s.KeyboardLocked {
		s.resetModifierState()
	}
	s.refreshTrayTip()
}

func (s *Service) toggleBlockedKey() {
	if !s.hasBlockedKey() {
		s.Config.Features.BlockedKeyEnabled = false
		s.refreshTrayTip()
		return
	}

	s.Config.Features.BlockedKeyEnabled = !s.Config.Features.BlockedKeyEnabled
	_ = saveAppConfig(s.ConfigPath, s.Config)
	s.refreshTrayTip()
}

func (s *Service) beginBlockedKeyCapture() {
	s.CaptureNextBlockedKey = true
	s.Config.Features.BlockedKeyEnabled = false
	s.refreshTrayTip()
}

func (s *Service) clearBlockedKey() {
	s.CaptureNextBlockedKey = false
	s.Config.Features.BlockedKeyEnabled = false
	s.Config.Features.BlockedVK = 0
	_ = saveAppConfig(s.ConfigPath, s.Config)
	s.refreshTrayTip()
}

func (s *Service) resetModifierState() {
	s.CtrlDown = false
	s.AltDown = false
	s.ShiftDown = false
}

func (s *Service) hasBlockedKey() bool {
	return s.Config.Features.BlockedVK != 0
}

func (s *Service) blockedKeyStatusLabel() string {
	if s.CaptureNextBlockedKey {
		return "Blocked Key: press any key..."
	}
	return fmt.Sprintf("Blocked Key: %s", vkName(s.Config.Features.BlockedVK))
}

func (s *Service) blockedKeyToggleLabel() string {
	if !s.hasBlockedKey() {
		return "Block Specific Key"
	}
	if s.Config.Features.BlockedKeyEnabled {
		return fmt.Sprintf("Block Specific Key: On (%s)", vkName(s.Config.Features.BlockedVK))
	}
	return fmt.Sprintf("Block Specific Key: Off (%s)", vkName(s.Config.Features.BlockedVK))
}

func (s *Service) blockedKeyCaptureLabel() string {
	if s.CaptureNextBlockedKey {
		return "Set Blocked Key: waiting..."
	}
	return "Set Blocked Key (Next Press)"
}

func (s *Service) blockedKeySummary() string {
	if s.CaptureNextBlockedKey {
		return "blocked key capture pending"
	}
	if !s.hasBlockedKey() {
		return "blocked key none"
	}
	state := "off"
	if s.Config.Features.BlockedKeyEnabled {
		state = "on"
	}
	return fmt.Sprintf("blocked key %s (%s)", vkName(s.Config.Features.BlockedVK), state)
}

func vkName(vk uint32) string {
	if vk == 0 {
		return "None"
	}

	named := map[uint32]string{
		0x08: "Backspace",
		0x09: "Tab",
		0x0D: "Enter",
		0x10: "Shift",
		0x11: "Ctrl",
		0x12: "Alt",
		0x13: "Pause",
		0x14: "CapsLock",
		0x1B: "Esc",
		0x20: "Space",
		0x21: "PageUp",
		0x22: "PageDown",
		0x23: "End",
		0x24: "Home",
		0x25: "Left",
		0x26: "Up",
		0x27: "Right",
		0x28: "Down",
		0x2D: "Insert",
		0x2E: "Delete",
		0x5B: "LeftWin",
		0x5C: "RightWin",
	}
	if name, ok := named[vk]; ok {
		return name
	}

	if vk >= 0x30 && vk <= 0x39 {
		return string(rune('0' + (vk - 0x30)))
	}
	if vk >= 0x41 && vk <= 0x5A {
		return string(rune('A' + (vk - 0x41)))
	}
	if vk >= 0x70 && vk <= 0x87 {
		return fmt.Sprintf("F%d", vk-0x6F)
	}
	if vk >= 0x60 && vk <= 0x69 {
		return fmt.Sprintf("Numpad%d", vk-0x60)
	}
	return fmt.Sprintf("VK_%d", vk)
}
