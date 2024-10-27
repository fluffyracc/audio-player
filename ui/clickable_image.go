package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

type ClickableInvisible struct {
	widget.BaseWidget
	onTap    func(event *fyne.PointEvent)
	onScroll func(event *fyne.ScrollEvent)
}

func NewClickableInvisible(onTap func(event *fyne.PointEvent), onScroll func(event *fyne.ScrollEvent)) *ClickableInvisible {
	c := &ClickableInvisible{
		onTap:    onTap,
		onScroll: onScroll,
	}
	// c.ExtendBaseWidget(c)
	return c
}

func (c *ClickableInvisible) Tapped(p *fyne.PointEvent) {
	c.onTap(p)
}

func (c *ClickableInvisible) Scrolled(e *fyne.ScrollEvent) {
	c.onScroll(e)
}
