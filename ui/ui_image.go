package ui

import (
	"audio-player/gtime"
	"golang.org/x/image/draw"
	"image"
	"log"
	"runtime"
	"strconv"
	"sync"
)

func (u *UI) getBaseRect() image.Rectangle {
	return image.Rect(u.baseImage.Bounds().Min.X, u.baseImage.Bounds().Min.Y, u.baseImage.Bounds().Max.X, u.baseImage.Bounds().Max.Y)
}

const useDrawPackage = false

// cropAndUpscaleImage returns a new image that is a cropped and upscaled version of the base image
func (u *UI) cropAndUpscaleImage() image.Image {
	gtime.Start("cropAndUpscaleImage")
	defer gtime.End("cropAndUpscaleImage")

	rect := u.getBaseRect()

	simg, ok := u.baseImage.(subImager)
	if !ok {
		log.Fatal("image does not support cropping")
	}

	gtime.Start("cropAndUpscaleImage.newImage")
	newImage := image.NewRGBA(image.Rect(0, 0, u.baseImage.Bounds().Max.X, u.baseImage.Bounds().Max.Y))
	gtime.End("cropAndUpscaleImage.newImage")
	zoomRound := float32(int(float32(rect.Dx()) * u.zoomLevel))

	if useDrawPackage {
		rect = image.Rect(0, 0, int(zoomRound), rect.Dy())

		// crop the image
		rect.Min.X = int(u.pan)
		rect.Max.X = int(zoomRound) + int(u.pan)

		cropImage := simg.SubImage(rect)

		// create a new image, copying the cropped stuff to the new one (upscaled)
		gtime.Start("cropAndUpscaleImage.draw")
		draw.NearestNeighbor.Scale(newImage, rect, cropImage, cropImage.Bounds(), draw.Over, nil)
		gtime.End("cropAndUpscaleImage.draw")

	} else {

		gtime.Start("cropAndUpscaleImage.crop")
		img := simg.SubImage(rect)
		gtime.End("cropAndUpscaleImage.crop")

		//newSizeX := img.Bounds().Max.X - img.Bounds().Min.X
		//newSizeY := img.Bounds().Max.Y - img.Bounds().Min.Y
		//log.Println("new resolution", strconv.Itoa(newSizeX)+"x"+strconv.Itoa(newSizeY))
		// create a new image, copying the cropped stuff to the new one (upscaled)

		gtime.Start("cropAndUpscaleImage.copy")

		cols := int((float32(u.baseImage.Bounds().Max.X) * u.zoomLevel) * u.getScale())
		rows := u.baseImage.Bounds().Max.Y
		// this is about the minimum you can go without it looking blurry.
		minCols := int(600 * u.getScale())
		if cols < minCols {
			cols = minCols
		}
		newImage = image.NewRGBA(image.Rect(0, 0, cols, u.baseImage.Bounds().Max.Y))

		log.Println("render at res", strconv.Itoa(cols)+"x"+strconv.Itoa(rows))

		var wg sync.WaitGroup
		numGoroutines := runtime.NumCPU() / 2
		rowsPerGoroutine := u.baseImage.Bounds().Max.Y / numGoroutines

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(startY, endY int) {
				defer wg.Done()
				for y := startY; y < endY; y++ {
					for x := 0; x < cols; x++ {
						nearX := int(zoomRound * float32(x) / float32(cols))
						if u.pan != 0 {
							nearX = nearX + int(u.pan)
						}
						newImage.Set(x, y, img.At(nearX, y))
					}
				}
			}(i*rowsPerGoroutine, (i+1)*rowsPerGoroutine)
		}
		wg.Wait()

		//newImage.Pix = bytes.Clone(imageResizeBuff)

		gtime.End("cropAndUpscaleImage.copy")
	}

	return newImage
}

func (u *UI) getCursorPosition() float32 {
	// position the cursor should be in, in pixels
	cursorPosition := float32(float32(u.s.Width) * u.playbackPercent)

	// zoomLevel is a percent so this is fine.
	if u.zoomLevel != 1 {
		cursorPosition = cursorPosition / u.zoomLevel
	}

	// pan is the offset from the left of the screen. it is *not* a percent.
	if u.pan != 0 {
		panPercent := (u.pan / u.zoomLevel) / float32(u.baseImage.Bounds().Max.X)
		cursorPosition = cursorPosition - (float32(u.s.Width) * panPercent)
	}

	return cursorPosition
}

func (u *UI) getCursorSeekPositionOnClick(posX, width float32) float32 {
	dur := u.a.Duration()

	// pan is the offset from the left of the screen. it is *not* a percent.
	if u.pan != 0 {
		panPercent := (u.pan / u.zoomLevel) / float32(u.baseImage.Bounds().Max.X)
		log.Println("pan at", panPercent)
		posX = posX + (float32(u.s.Width) * panPercent)
	}

	percent := posX / width // percent on screen
	if u.zoomLevel == 1 {
		return percent * dur
	}

	// zoomLevel is a percent so this is fine.
	percent = percent * u.zoomLevel

	return percent * dur
}
