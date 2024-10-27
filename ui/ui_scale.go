package ui

import (
	"os"
	"strconv"
)

func (u *UI) getScale() float32 {
	str := os.Getenv("FYNE_SCALE")
	if str != "" {
		num, _ := strconv.ParseFloat(str, 32)
		if num > 0.1 && num < 10 {
			return float32(num)
		}
	}

	return 1
}
