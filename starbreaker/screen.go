package starbreaker

import (
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"github.com/xdung24/games/internal/browserinput"
)

// Screen owns a game session and preserves its progress between menu visits.
type Screen struct {
	window  fyne.Window
	game    *Game
	board   *Board
	content fyne.CanvasObject
}

// NewScreen creates a game session, initially hidden until Show is called.
func NewScreen(w fyne.Window, preferences fyne.Preferences, onBack func()) *Screen {
	game := NewGame()
	sound := newSoundPlayer()
	sound.SetMuted(preferences.Bool("starbreaker.sound.muted"))
	game.onSound = sound.Play
	board := NewBoard(game)
	board.muted = sound.IsMuted
	board.toggleMute = func() bool {
		muted := sound.ToggleMuted()
		preferences.SetBool("starbreaker.sound.muted", muted)
		return muted
	}
	s := &Screen{window: w, game: game, board: board, content: board}
	if height := browserinput.ControlsHeight(); height > 0 {
		space := canvas.NewRectangle(color.Transparent)
		space.SetMinSize(fyne.NewSize(0, height))
		s.content = container.NewBorder(nil, space, nil, nil, board)
	}
	board.onBack = func() {
		s.Hide()
		onBack()
	}
	board.Hide()
	startLoop(game, board)
	return s
}

// Show opens the game and restores keyboard focus.
func (s *Screen) Show() {
	s.board.Show()
	s.window.SetContent(s.content)
	s.window.Canvas().Focus(s.board)
	if s.board.renderer != nil {
		s.board.renderer.bgAnim.Stop()
		s.board.renderer.bgAnim.Start()
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
	s.board.Hide()
	if s.board.renderer != nil {
		s.board.renderer.bgAnim.Stop()
	}
}

func startLoop(game *Game, board *Board) {
	go func() {
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		last := time.Now()
		for now := range ticker.C {
			dt := min(float32(now.Sub(last).Seconds()), 0.05)
			last = now
			fyne.Do(func() {
				if board.Visible() && game.Tick(dt) {
					board.Refresh()
				}
			})
		}
	}()
}
