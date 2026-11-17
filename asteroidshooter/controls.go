package asteroidshooter

import (
	"fyne.io/fyne/v2"
	"github.com/xdung24/games/internal/touchcontrols"
)

type heldControls struct {
	left, right, thrust, fire bool
}

func (b *Board) syncHeld() {
	b.game.leftHeld = b.keys.left || b.touch.left
	b.game.rightHeld = b.keys.right || b.touch.right
	b.game.thrustHeld = b.keys.thrust || b.touch.thrust
	b.game.fireHeld = b.keys.fire || b.touch.fire
}

func (b *Board) pause() {
	b.game.TogglePause()
	b.FocusLost()
	b.Refresh()
}

func (b *Board) restart() {
	b.FocusLost()
	b.game.StartGame()
	b.Refresh()
}

func (b *Board) mute() {
	b.escape.Reset()
	if b.toggleMute != nil {
		b.toggleMute()
		b.Refresh()
	}
}

func (b *Board) pressFire(touch bool) {
	b.escape.Reset()
	b.game.Activate()
	if touch {
		b.touch.fire = b.game.state == StatePlaying
	} else {
		b.keys.fire = b.game.state == StatePlaying
	}
	b.syncHeld()
	b.game.Fire()
	b.Refresh()
}

func (s *Screen) newControls(content fyne.CanvasObject) *touchcontrols.Panel {
	b := s.board
	held := func(name string, icon fyne.Resource, state *bool) *touchcontrols.Button {
		return touchcontrols.NewButton(name, icon, func() {
			b.escape.Reset()
			*state = true
			b.syncHeld()
		}, func() {
			*state = false
			b.syncHeld()
		})
	}
	left := held("Left", touchcontrols.LeftIcon, &b.touch.left)
	right := held("Right", touchcontrols.RightIcon, &b.touch.right)
	thrust := held("Thrust", touchcontrols.ThrustIcon, &b.touch.thrust)
	fire := touchcontrols.NewButton("Fire / Play", touchcontrols.FireIcon, func() {
		b.pressFire(true)
	}, func() {
		b.touch.fire = false
		b.syncHeld()
	})
	pause := touchcontrols.NewButton("Pause", touchcontrols.PauseIcon, b.pause, nil)
	mute := touchcontrols.NewButton("Mute", touchcontrols.MuteIcon, b.mute, nil)
	panel := touchcontrols.NewPanel(content,
		[]*touchcontrols.Button{left, right},
		nil,
		[]*touchcontrols.Button{thrust, fire},
		[]*touchcontrols.Button{pause, mute},
	)
	panel.Overlay = true
	return panel
}
