package croakcross

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// Screen owns a game session and preserves its progress between menu visits.
type Screen struct {
	window  fyne.Window
	game    *Game
	board   *Board
	content fyne.CanvasObject
}

// NewScreen creates one CroakCross session, initially hidden until Show is called.
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

	screen := &Screen{window: w, game: game, board: board}
	board.onBack = func() {
		screen.Hide()
		onBack()
	}
	backButton := widget.NewButtonWithIcon("Home", theme.NavigateBackIcon(), board.onBack)
	backButton.Importance = widget.LowImportance
	overlay := container.NewVBox(layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), backButton, mutedIcon))
	screen.content = container.NewStack(board, overlay)
	board.Hide()
	startLoop(game, board)
	return screen
}

// Show opens the game and restores its keyboard focus.
func (s *Screen) Show() {
	s.board.Show()
	s.window.SetContent(s.content)
	s.window.Canvas().Focus(s.board)
}

// Pause freezes play when the application leaves the foreground.
func (s *Screen) Pause() {
	s.game.Pause()
	if s.board.Visible() {
		s.board.Refresh()
	}
}

// Hide pauses the session before returning to the menu.
func (s *Screen) Hide() {
	s.Pause()
	s.board.FocusLost()
	s.board.Hide()
}
