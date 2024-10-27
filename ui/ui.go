package ui

import (
	"audio-player/audio"
	"audio-player/gtime"
	"audio-player/visu"
	"bytes"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"image"
	"image/color"
	"log"
	"path"
	"sync"
	"sync/atomic"
)

type UI struct {
	audioFile string

	a   *audio.Audio
	w   fyne.Window
	s   *LayoutMain
	app fyne.App

	baseImage image.Image
	image     *canvas.Image

	label       *widget.Label
	cursor      fyne.CanvasObject
	pauseButton *widget.Button
	pausePos    float32

	zoomLevel   float32
	pan         float32
	scrollEvent atomic.Bool

	playState       int
	playStateMux    sync.Mutex
	width           float32
	duration        float32
	playbackPercent float32

	// keys
	controlPressed bool
}

func (u *UI) playAudioSync(pos float32, dur float32) {
	// play
	if err := u.a.Start(float64(pos)); err != nil {
		log.Println("error starting audio:", err)
	}
	// setup cursor render
	u.playCursor(pos, dur)
}

func (u *UI) pauseAudioSync(pos float32, dur float32) float32 {
	u.pauseCursor(pos, dur)
	u.a.Stop()
	return u.playbackPercent
}

func New() *UI {
	u := &UI{
		zoomLevel: 1,
		pan:       0,
	}
	return u
}

func (u *UI) togglePlay() {
	if u.pausePos == 0 {
		u.pauseAudioSync(u.playbackPercent*u.a.Duration(), u.a.Duration())
		u.pausePos = u.playbackPercent * u.a.Duration()
		u.pauseButton.SetText("Resume")
	} else {
		u.playAudioSync(u.pausePos, u.a.Duration())
		u.pausePos = 0
		u.pauseButton.SetText("Pause")
	}
}

type subImager interface {
	SubImage(r image.Rectangle) image.Image
}

func (u *UI) Run(audioFile string) error {
	gtime.Start("ui.Run")
	// before start, clean up old stuff
	if u.a != nil {
		u.a.Stop()
	}

	u.audioFile = audioFile
	u.a = audio.New(u.audioFile)
	u.s = &LayoutMain{}

	gtime.Start("ui.Run.createAppAndWindow")
	didMakeApp := false
	if u.app == nil {
		u.app = app.New()
		didMakeApp = true
	}

	titleStr := path.Base(u.audioFile)
	if u.w == nil {
		u.w = u.app.NewWindow(titleStr)
		u.w.Resize(fyne.NewSize(800, 200))
	} else {
		u.w.SetTitle(titleStr)
	}

	u.w.Canvas().SetOnTypedKey(func(event *fyne.KeyEvent) {
		if event.Name == fyne.KeySpace {
			u.togglePlay()
		}
	})
	gtime.End("ui.Run.createAppAndWindow")

	gtime.Start("ui.Run.CreateElements")
	var ct *fyne.Container
	u.cursor = canvas.NewRectangle(color.White)
	imageWidth, imageHeight := visu.GetSize()

	u.label = widget.NewLabel("0:00:00 / 0:00:00")
	u.label.Alignment = fyne.TextAlignCenter

	// TODO: would be nice if placeholder was a bit more fancy (loading animation? placeholder wave form?)
	u.baseImage = image.NewAlpha(image.Rect(0, 0, imageWidth, imageHeight))
	u.image = canvas.NewImageFromImage(u.baseImage)

	go (func() {
		imageBits, err := visu.GenerateImage(u.audioFile)
		if err != nil {
			log.Fatal("error generating image:", err)
		}
		decodedImage, _, err := image.Decode(bytes.NewReader(imageBits))
		if err != nil {
			log.Fatal("error decoding image:", err)
		}
		u.baseImage = decodedImage
		u.image.Image = decodedImage
		u.image.Refresh()
	})()

	//image := canvas.NewImageFromReader(bytes.NewReader(imageBits), "waveform_"+u.audioFile)
	cursorPositioner := NewClickableInvisible(func(event *fyne.PointEvent) {
		// translate to position
		pos := u.getCursorSeekPositionOnClick(event.Position.X, float32(u.image.Size().Width))
		dur := u.a.Duration()

		if u.pausePos != 0 {
			u.pausePos = pos
			u.playbackPercent = pos / dur
			u.pauseCursor(pos, dur)
		} else {
			u.playAudioSync(pos, dur)
		}
	}, func(event *fyne.ScrollEvent) {
		if !u.scrollEvent.Swap(true) {
			return
		}
		defer u.scrollEvent.Swap(false)

		n := -event.Scrolled.DY
		mode := "pan"
		if u.controlPressed {
			mode = "zoom"
		}

		oldZoom := u.zoomLevel
		oldPan := u.pan

		if mode == "zoom" {
			percentToInc := n * 0.001
			newZoom := u.zoomLevel + percentToInc
			if newZoom < 0.01 {
				newZoom = 0.01
			} else if newZoom > 1 {
				newZoom = 1
			}

			u.zoomLevel = newZoom
		} else if mode == "pan" {
			percentToInc := n
			newPos := u.pan + percentToInc
			maxPan := float32(u.baseImage.Bounds().Max.X)
			if newPos < 0 {
				newPos = 0
			} else if newPos > maxPan {
				newPos = maxPan
			}

			u.pan = newPos
		}

		if oldZoom != u.zoomLevel || oldPan != u.pan {

			u.image.Image = u.cropAndUpscaleImage()
			u.image.Refresh()
			// update the cursor position too
			if u.pausePos != 0 {
				u.pauseCursor(u.pausePos, u.a.Duration())
			}
		}
	})

	replayButton := widget.NewButton("Replay", func() {
		// reset pause position/label
		u.pausePos = 0
		u.pauseButton.SetText("Pause")
		u.playAudioSync(0, u.a.Duration())
	})

	u.pauseButton = widget.NewButton("Pause", func() {
		u.togglePlay()
	})

	closeButton := widget.NewButton("Close", func() {
		u.a.Stop()
		u.app.Quit()
	})

	ct = container.New(u.s, u.image, cursorPositioner, u.cursor, u.label, replayButton, u.pauseButton, closeButton)

	gtime.End("ui.Run.CreateElements") // 1ms

	go (func() {
		// u.a.Duration() is called before goroutine is created, so we can't just do "go playAudioSync(...)"
		u.playAudioSync(0, u.a.Duration())
	})()

	u.w.SetContent(ct)

	if deskCanvas, ok := u.w.Canvas().(desktop.Canvas); ok {
		deskCanvas.SetOnKeyDown(func(key *fyne.KeyEvent) {
			if key.Name == "LeftControl" {
				u.controlPressed = true
			}
		})
		deskCanvas.SetOnKeyUp(func(key *fyne.KeyEvent) {
			if key.Name == "LeftControl" {
				u.controlPressed = false
			}
		})
	}

	if didMakeApp {
		gtime.End("ui.Run")
		gtime.End("main")
		u.w.ShowAndRun()
		log.Println("app exit")
		u.a.Stop()
	} else {
		// new song playing, so focus window
		u.w.RequestFocus()
	}

	return nil
}
