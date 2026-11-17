package flappygophy

import (
	"fmt"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type Screen struct {
	window            fyne.Window
	game              *Game
	scene             *gameScene
	input             *tapWidget
	content           *fyne.Container
	textBox           *fyne.Container
	back              *widget.Button
	pause             *widget.Button
	lastState         State
	lastScore, lastHi int
}

func NewScreen(w fyne.Window, preferences fyne.Preferences, onBack func()) *Screen {
	initAudio()
	s := newScreen(w, NewGame(preferences), onBack)
	go func() {
		ticker := time.NewTicker(time.Second / 60)
		defer ticker.Stop()
		for range ticker.C {
			fyne.Do(func() {
				if !s.content.Visible() {
					return
				}
				if s.game.update() {
					s.refresh()
				}
			})
		}
	}()
	return s
}

func newScreen(w fyne.Window, g *Game, onBack func()) *Screen {
	s := &Screen{window: w, game: g, lastState: State(-1), lastScore: -1, lastHi: -1}
	s.scene = newGameScene(g)
	s.input = newTapWidget(func() {
		g.flap()
		s.refresh()
	})
	s.input.onPause = s.Pause
	s.input.onBack = func() {
		s.Hide()
		onBack()
	}
	s.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), s.input.onBack)
	s.pause = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), func() {
		s.input.escape.Reset()
		g.TogglePause()
		w.Canvas().Focus(s.input)
		s.refresh()
	})

	scoreBG := canvas.NewRectangle(color.RGBA{R: 185, G: 225, B: 250, A: 255})
	scoreBG.CornerRadius = 4
	scoreBG.StrokeWidth = 2
	scoreBG.StrokeColor = color.RGBA{R: 145, G: 133, B: 118, A: 255}
	s.textBox = container.NewStack(scoreBG, container.NewPadded(g.scoreText))
	top := container.New(layout.NewCustomPaddedLayout(10, 0, 0, 0), container.NewCenter(s.textBox))
	bottom := container.NewPadded(container.NewHBox(s.back, layout.NewSpacer(), s.pause))
	s.content = container.NewStack(
		container.New(outOfSafeLayout(w), container.NewClip(s.scene)),
		s.input,
		container.NewBorder(top, bottom, nil, nil),
		container.NewCenter(container.NewVBox(g.msgText, g.subText)),
	)
	s.refresh()
	s.content.Hide()
	return s
}

func (s *Screen) Show() {
	s.content.Show()
	s.window.SetContent(s.content)
	s.window.Canvas().Focus(s.input)
	s.refresh()
}

func (s *Screen) Pause() {
	s.game.Pause()
	if s.content.Visible() {
		s.refresh()
	}
}

func (s *Screen) Hide() {
	s.Pause()
	s.input.FocusLost()
	s.content.Hide()
}

func (s *Screen) refresh() {
	g := s.game
	g.mu.Lock()
	state, score, hi := g.state, g.score, g.hiScore
	g.mu.Unlock()

	if score != s.lastScore {
		g.scoreText.Text = fmt.Sprintf("%d", score)
		g.scoreText.Refresh()
		s.lastScore = score
	}
	if state != s.lastState || hi != s.lastHi {
		s.lastState, s.lastHi = state, hi
		s.textBox.Show()
		g.msgText.Show()
		g.subText.Show()
		switch state {
		case StateStart:
			s.textBox.Hide()
			g.msgText.Text = "FLAPPY GOPHY"
			g.subText.Text = "hit space/enter to play"
			if fyne.CurrentDevice().IsMobile() {
				g.subText.Text = "tap to play"
			}
		case StatePlaying:
			g.msgText.Hide()
			g.subText.Hide()
		case StatePaused:
			g.msgText.Text = "PAUSED"
			g.subText.Text = "Tap/Space to resume; Esc for Home"
		case StateOver:
			g.msgText.Text = "GAME OVER"
			g.subText.Text = fmt.Sprintf("Best: %d  /  Score: %d - tap to retry", hi, score)
		}
		g.msgText.Refresh()
		g.subText.Refresh()
		pauseIcon := theme.MediaPauseIcon()
		if state == StatePaused {
			pauseIcon = theme.MediaPlayIcon()
		}
		s.pause.SetIcon(pauseIcon)
		if state == StatePlaying || state == StatePaused {
			s.pause.Enable()
		} else {
			s.pause.Disable()
		}
	}
	s.scene.refresh(time.Now())
}
