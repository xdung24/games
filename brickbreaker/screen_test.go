package brickbreaker

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestBoardEscapeReturnsHome(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	for _, state := range []GameState{StateReady, StatePlaying, StatePaused, StateGameOver, StateWon} {
		game := newTestGame()
		game.state = state
		game.score = 120
		board := NewBoard(game)
		w := test.NewWindow(board)
		screen := &Screen{window: w, game: game, board: board, content: board}
		home := widget.NewLabel("Home")
		backs := 0
		board.onBack = func() {
			screen.Hide()
			backs++
			w.SetContent(home)
		}
		screen.Show()
		escape := &fyne.KeyEvent{Name: fyne.KeyEscape}
		board.TypedKey(escape)
		want := state
		if state == StatePlaying {
			want = StatePaused
		}
		if game.state != want || backs != 0 || !board.Visible() {
			t.Fatalf("first Esc must pause without leaving the game in state %v", state)
		}
		board.KeyUp(escape)
		board.TypedKey(escape)
		if backs != 1 || board.Visible() || w.Content() != home || game.score != 120 {
			t.Fatalf("second Esc must return Home and preserve progress in state %v", state)
		}
		screen.Show()
		board.TypedKey(escape)
		if backs != 1 {
			t.Fatal("returning to a game must start a fresh Esc sequence")
		}
		w.Close()
	}
}

func TestBoardBackControl(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	game := newTestGame()
	board := NewBoard(game)
	backs := 0
	board.onBack = func() { backs++ }
	w := test.NewWindow(board)
	defer w.Close()
	for _, size := range []fyne.Size{fyne.NewSize(320, 700), fyne.NewSize(640, 600)} {
		w.Resize(size)
		board.Refresh()
		r := board.renderer
		if r.back.Position().X+r.back.Size().Width >= r.score.Position().X {
			t.Fatal("back control must sit to the left of SCORE without overlap")
		}
		if r.score.Position().X+r.score.MinSize().Width >= (game.w-r.levelText.MinSize().Width)/2 {
			t.Fatalf("score overlaps the level at size %v", size)
		}
	}
	r := board.renderer
	test.TapCanvas(w.Canvas(), r.back.Position().Add(fyne.NewPos(5, 5)))
	if backs != 1 || game.state != StateReady {
		t.Fatal("back control must return to the menu without activating the game")
	}
	game.Launch()
	game.Pause()
	board.Refresh()
	test.TapCanvas(w.Canvas(), r.back.Position().Add(fyne.NewPos(5, 5)))
	if backs != 2 || game.state != StatePaused {
		t.Fatal("back control must remain usable above the pause overlay")
	}
}

func TestScreenPreservesGameWhenHidden(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	game := newTestGame()
	board := NewBoard(game)
	w := test.NewWindow(board)
	defer w.Close()
	screen := &Screen{window: w, game: game, board: board, content: board}
	screen.Show()
	if w.Canvas().Focused() != board {
		t.Fatal("showing the game must restore keyboard focus")
	}
	game.Launch()
	game.score = 120
	game.leftHeld = true
	game.rightHeld = true
	screen.Hide()
	if board.Visible() || game.state != StatePaused || game.leftHeld || game.rightHeld {
		t.Fatal("hiding the game must pause it, hide the board, and release held controls")
	}
	screen.Show()
	if !board.Visible() || game.state != StatePaused || game.score != 120 {
		t.Fatal("reopening the game must preserve paused progress")
	}
	game.TogglePause()
	game.leftHeld, game.rightHeld = true, true
	screen.Pause()
	if game.state != StatePaused || game.leftHeld || game.rightHeld {
		t.Fatal("losing foreground must pause the game and release held controls")
	}
}
