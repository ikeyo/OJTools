//go:build windows

package remap

import (
	"os"

	"ojtools/internal/win32"
)

func createTrayIcon() (win32.HICON, error) {
	exePath, err := os.Executable()
	if err == nil {
		icon, err := win32.ExtractSmallIcon(exePath)
		if err == nil {
			return icon, nil
		}
	}

	icon, err := win32.LoadIcon(win32.IDI_APPLICATION)
	if err == nil {
		return icon, nil
	}

	pattern := []string{
		"................",
		"................",
		"................",
		"................",
		"..#####..#####..",
		"..#www#..#www#..",
		"..#www#..#www#..",
		"..#www#..#www#..",
		"..#####..#####..",
		"................",
		"................",
		"................",
		"................",
		"................",
		"................",
		"................",
	}

	andBits := make([]byte, 32)
	xorBits := make([]byte, 32)
	for y, row := range pattern {
		for x, cell := range row {
			byteIndex := y*2 + x/8
			bit := byte(1 << (7 - (x % 8)))

			switch cell {
			case '.':
				andBits[byteIndex] |= bit
			case '#':
				// black pixel: and=0 xor=0
			case 'w':
				xorBits[byteIndex] |= bit
			default:
				andBits[byteIndex] |= bit
			}
		}
	}

	return win32.CreateIcon(16, 16, 1, 1, andBits, xorBits)
}
