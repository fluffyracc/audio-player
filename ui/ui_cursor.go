package ui

import (
	"fyne.io/fyne/v2"
	"log"
	"time"
)

func (u *UI) playCursor(pos, dur float32) {
	playStart := time.Now()
	u.playStateMux.Lock()
	u.playState++
	expect := u.playState
	u.playStateMux.Unlock()

	durationStr := u.s.FormatDuration(dur)

	go (func() {
		origPos := pos
		for pos <= dur {
			u.playStateMux.Lock()
			if u.playState != expect {
				u.playStateMux.Unlock()
				//log.Println("LayoutMain playback interrupted")
				return
			}
			u.playbackPercent = pos / dur
			//percent := u.playbackPercent
			u.playStateMux.Unlock()

			amountToAdd := float32(time.Since(playStart).Milliseconds()) / 1000
			pos = origPos + amountToAdd

			u.cursor.Move(fyne.NewPos(u.getCursorPosition(), 2))
			u.label.SetText(u.s.FormatDuration(pos) + " / " + durationStr)

			time.Sleep(time.Millisecond * 10)
		}
		log.Println("LayoutMain playback finished")
	})()
}

func (u *UI) pauseCursor(pos, dur float32) {
	u.playStateMux.Lock()
	u.playState = u.playState + 1
	u.playbackPercent = pos / dur
	u.playStateMux.Unlock()

	u.cursor.Move(fyne.NewPos(u.getCursorPosition(), 2))
	u.label.SetText(u.s.FormatDuration(pos) + " / " + u.s.FormatDuration(dur))
}
