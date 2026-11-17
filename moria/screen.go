package moria

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const prefKeyMuted = "sound.muted"

type Screen struct {
	window  fyne.Window
	game    *Game
	board   *Board
	content fyne.CanvasObject
}

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
	var muteButton *widget.Button
	board.onToggleMute = func() {
		muted := !sound.Muted()
		sound.SetMuted(muted)
		preferences.SetBool(prefKeyMuted, muted)
		if muted {
			mutedIcon.Show()
			muteButton.SetText("Unmute (M)")
		} else {
			mutedIcon.Hide()
			muteButton.SetText("Mute (M)")
		}
	}

	screen := &Screen{window: w, game: game, board: board}
	board.onBack = func() {
		screen.Hide()
		onBack()
	}
	backButton := widget.NewButtonWithIcon("Home", theme.NavigateBackIcon(), board.onBack)
	backButton.Importance = widget.LowImportance
	muteButton = widget.NewButton("Mute (M)", board.onToggleMute)
	if sound.Muted() {
		muteButton.SetText("Unmute (M)")
	}
	muteButton.Importance = widget.LowImportance
	overlay := container.NewVBox(layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), muteButton, backButton, mutedIcon))
	screen.content = container.NewStack(board, overlay)
	board.Hide()
	startLoop(game, board)
	return screen
}

func (s *Screen) Show() {
	s.board.Show()
	s.window.SetContent(s.content)
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

func startLoop(game *Game, board *Board) {
	go func() {
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		last := time.Now()
		for now := range ticker.C {
			dt := float32(now.Sub(last).Seconds())
			last = now
			if dt > 0.05 {
				dt = 0.05
			}
			fyne.Do(func() {
				if !board.Visible() {
					return
				}
				game.Update(dt, board.frameInput())
				board.Refresh()
			})
		}
	}()
}
