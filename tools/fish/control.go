package fish

import (
	"math"

	"github.com/lxn/win"

	"yhbox/pkg/input"
)

type fishingControl struct {
	controlDir int
}

func newFishingControl() *fishingControl {
	return &fishingControl{}
}

func (fc *fishingControl) reset() {
	fc.controlDir = 0
}

func chooseDirection(err float64, deadzonePx float64) int {
	if math.Abs(err) <= deadzonePx {
		return 0
	}
	if err > 0 {
		return 1
	}
	return -1
}

func (fc *fishingControl) applyDirection(hwnd win.HWND, dir int) {
	if dir > 0 {
		dir = 1
	} else if dir < 0 {
		dir = -1
	}
	if dir == fc.controlDir {
		return
	}
	fc.controlDir = dir
	switch dir {
	case 1:
		input.KeyUp(hwnd, "a")
		input.KeyDown(hwnd, "d")
	case -1:
		input.KeyUp(hwnd, "d")
		input.KeyDown(hwnd, "a")
	default:
		input.ReleaseAll(hwnd)
	}
}
