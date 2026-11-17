package flappychick

import (
	"time"

	"fyne.io/fyne/v2"
)

const bestKey = "flappychick.best"

type Screen struct {
	window fyne.Window
	game   *Game
	board  *Board
}

func NewScreen(w fyne.Window, preferences fyne.Preferences, onBack func()) *Screen {
	g := NewGame()
	g.best = preferences.Int(bestKey)
	sound := newSoundPlayer()
	sound.muted = preferences.Bool("sound.muted")
	g.onSound = sound.Play
	b := NewBoard(g)
	s := &Screen{window: w, game: g, board: b}
	b.onBack = func() {
		s.Hide()
		onBack()
	}
	b.onMute = func() {
		sound.muted = !sound.muted
		preferences.SetBool("sound.muted", sound.muted)
		b.muted = sound.muted
		b.Refresh()
	}
	b.muted = sound.muted
	b.Hide()
	go func() {
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		last := time.Now()
		for now := range ticker.C {
			dt := min(float32(now.Sub(last).Seconds()), 0.05)
			last = now
			fyne.Do(func() {
				if !b.Visible() {
					return
				}
				best := g.best
				if g.Tick(dt) {
					if g.best > best {
						preferences.SetInt(bestKey, g.best)
					}
					b.Refresh()
				}
			})
		}
	}()
	return s
}

func (s *Screen) Show() {
	s.board.Show()
	s.window.SetContent(s.board)
	s.window.Canvas().Focus(s.board)
}

func (s *Screen) Pause() {
	s.game.Pause()
	if s.board.Visible() {
		s.board.Refresh()
	}
}

func (s *Screen) Hide() {
	s.Pause()
	s.board.FocusLost()
	s.board.Hide()
}
