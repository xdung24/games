// game — a nostalgic single-player breakout clone built with Fyne.
//
// Steer the paddle with the mouse, a finger, or the arrow keys (A/D also work),
// keep the ball in play, and break as many ocean-blue blocks as you can. Clearing
// the board advances you to a faster level; the game ends when your lives run out.
//
// Controls:
//
//	Mouse / finger / ← → / A D  move the paddle
//	Tap / click / Space         launch the ball · pause/resume · play again
//	P                           pause / resume
//	R                           restart
//	M                           mute / unmute the sound
//
// Play also pauses automatically whenever the app leaves the foreground.
package brickbreaker

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/xdung24/games/internal/touchcontrols"
)

const targetFPS = 60

const prefKeyMuted = "sound.muted"

// Screen owns the game view and preserves its progress between menu visits.
type Screen struct {
	window   fyne.Window
	game     *Game
	board    *Board
	content  fyne.CanvasObject
	controls *touchcontrols.Panel
}

// NewScreen creates one game session, initially hidden until Show is called.
func NewScreen(w fyne.Window, preferences fyne.Preferences, onBack func()) *Screen {
	game := NewGame()
	sound := newSoundPlayer()
	sound.SetMuted(preferences.Bool(prefKeyMuted))
	game.onSound = sound.Play
	board := NewBoard(game)

	mutedIcon := widget.NewIcon(theme.VolumeMuteIcon())
	if !sound.Muted() {
		mutedIcon.Hide()
	}
	board.onToggleMute = func() {
		muted := !sound.Muted()
		sound.SetMuted(muted)
		preferences.SetBool(prefKeyMuted, muted)
		if muted {
			mutedIcon.Show()
		} else {
			mutedIcon.Hide()
		}
	}

	overlay := container.NewVBox(layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), mutedIcon))
	s := &Screen{
		window: w, game: game, board: board,
		content: container.NewStack(board, overlay),
	}
	if touchcontrols.Enabled() {
		s.controls = s.newControls(s.content)
		s.content = s.controls
		board.onFocusLost = s.controls.Reset
	}
	board.onBack = func() {
		s.Hide()
		onBack()
	}
	board.Hide()
	startLoop(game, board)
	return s
}

// Show opens the game and restores its keyboard focus.
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

// Hide pauses the session and releases held controls before returning to a menu.
func (s *Screen) Hide() {
	s.Pause()
	if s.controls != nil {
		s.controls.Detach()
	}
	s.board.FocusLost()
	s.board.Hide()
}

// startLoop drives the simulation on a fixed-rate ticker. All state changes and
// the canvas refresh are marshalled onto the UI goroutine via fyne.Do, so the
// model is only ever touched from one thread and needs no locking.
func startLoop(game *Game, board *Board) {
	go func() {
		ticker := time.NewTicker(time.Second / targetFPS)
		defer ticker.Stop()
		last := time.Now()
		for now := range ticker.C {
			dt := float32(now.Sub(last).Seconds())
			last = now
			if dt > 0.05 { // clamp after a stall so the ball never teleports
				dt = 0.05
			}
			fyne.Do(func() {
				if !board.Visible() {
					return
				}
				// Do the least work the frame needs: nothing when idle, a cheap
				// ball/paddle reposition while the ball flies, and a full redraw
				// only when something structural (bricks, score, banner) changed.
				switch game.Tick(dt) {
				case RenderMove:
					board.MoveDynamic()
				case RenderFull:
					board.Refresh()
				}
			})
		}
	}()
}
