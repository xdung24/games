package brickbreaker

import (
	"fyne.io/fyne/v2"
	"github.com/xdung24/games/internal/touchcontrols"
)

type heldControls struct {
	left, right bool
}

func (b *Board) syncHeld() {
	b.game.leftHeld = b.keys.left || b.touch.left
	b.game.rightHeld = b.keys.right || b.touch.right
}

func (s *Screen) newControls(content fyne.CanvasObject) *touchcontrols.Panel {
	b := s.board
	left := touchcontrols.NewButton("Left", touchcontrols.LeftIcon, func() {
		b.escape.Reset()
		b.touch.left = true
		b.syncHeld()
	}, func() {
		b.touch.left = false
		b.syncHeld()
	})
	right := touchcontrols.NewButton("Right", touchcontrols.RightIcon, func() {
		b.escape.Reset()
		b.touch.right = true
		b.syncHeld()
	}, func() {
		b.touch.right = false
		b.syncHeld()
	})
	left.Translucent, right.Translucent = true, true
	panel := touchcontrols.NewPanel(content, []*touchcontrols.Button{left}, nil, []*touchcontrols.Button{right}, nil)
	panel.Overlay = true
	panel.ControlsY = 0.8
	return panel
}
