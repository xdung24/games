package brickbreaker

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

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
	screen.Pause()
	if game.state != StatePaused {
		t.Fatal("losing foreground must pause the game")
	}
}
