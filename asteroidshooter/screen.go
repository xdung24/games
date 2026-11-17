package asteroidshooter

import (
	"time"

	"fyne.io/fyne/v2"
	"github.com/xdung24/games/internal/touchcontrols"
)

// Screen owns a game session and preserves its progress between menu visits.
type Screen struct {
	window   fyne.Window
	game     *Game
	board    *Board
	content  fyne.CanvasObject
	controls *touchcontrols.Panel
}

// NewScreen creates a game session, initially hidden until Show is called.
func NewScreen(w fyne.Window, preferences fyne.Preferences, onBack func()) *Screen {
	game := NewGame()
	sound := newSoundPlayer()
	sound.SetMuted(preferences.Bool("asteroidshooter.sound.muted"))
	game.onSound = sound.Play
	board := NewBoard(game)
	board.muted = sound.IsMuted
	board.toggleMute = func() bool {
		muted := sound.ToggleMuted()
		preferences.SetBool("asteroidshooter.sound.muted", muted)
		return muted
	}
	s := &Screen{window: w, game: game, board: board, content: board}
	if touchcontrols.Enabled() {
		s.controls = s.newControls(board)
		s.content = s.controls
		board.onFocusLost = s.controls.Reset
	}
	board.onBack = func() {
		s.Hide()
		onBack()
	}
	board.Hide()
	startLoop(board)
	return s
}

// Show opens the game and restores keyboard focus.
func (s *Screen) Show() {
	s.board.Show()
	s.window.SetContent(s.content)
	s.window.Canvas().Focus(s.board)
	if s.controls != nil {
		s.controls.Attach()
	}
}

// Pause freezes play when the application leaves the foreground.
func (s *Screen) Pause() {
	s.game.Pause()
	s.board.FocusLost()
	if s.board.Visible() {
		s.board.Refresh()
	}
}

// Hide pauses the session before returning to the menu.
func (s *Screen) Hide() {
	s.Pause()
	if s.controls != nil {
		s.controls.Detach()
	}
	s.board.Hide()
}

func (b *Board) tick(dt float32) {
	if !b.Visible() || !b.game.Tick(dt) {
		return
	}
	if b.game.state == StatePlaying || b.game.state == StateAttract {
		b.backgroundTime += dt
	}
	b.Refresh()
}

func startLoop(board *Board) {
	go func() {
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		last := time.Now()
		for now := range ticker.C {
			dt := min(float32(now.Sub(last).Seconds()), 0.05)
			last = now
			fyne.Do(func() {
				board.tick(dt)
			})
		}
	}()
}
