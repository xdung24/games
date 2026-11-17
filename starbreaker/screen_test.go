package starbreaker

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestScreenPreservesGameWhenHidden(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	game := newPlayingGame()
	board := NewBoard(game)
	w := test.NewWindow(board)
	defer w.Close()
	screen := &Screen{window: w, game: game, board: board, content: board}
	screen.Show()
	if w.Canvas().Focused() != board {
		t.Fatal("showing the game must restore keyboard focus")
	}
	game.score = 120
	for _, key := range []fyne.KeyName{fyne.KeyLeft, fyne.KeyRight, fyne.KeyUp, fyne.KeySpace} {
		board.KeyDown(&fyne.KeyEvent{Name: key})
	}
	game.ship.thrust = true
	screen.Hide()
	if board.Visible() || game.state != StatePaused {
		t.Fatal("hiding the game must pause it and hide the board")
	}
	if game.leftHeld || game.rightHeld || game.thrustHeld || game.fireHeld || game.ship.thrust {
		t.Fatal("hiding the game must release all held controls")
	}
	screen.Show()
	if !board.Visible() || game.state != StatePaused || game.score != 120 {
		t.Fatal("reopening the game must preserve paused progress")
	}
	game.TogglePause()
	board.KeyDown(&fyne.KeyEvent{Name: fyne.KeyUp})
	screen.Pause()
	if game.state != StatePaused || game.thrustHeld {
		t.Fatal("losing foreground must pause the game and release held controls")
	}
}

func TestBoardFocusLostReleasesControls(t *testing.T) {
	game := newPlayingGame()
	board := NewBoard(game)
	game.leftHeld, game.rightHeld = true, true
	game.thrustHeld, game.fireHeld = true, true
	game.ship.thrust = true
	board.FocusLost()
	if game.leftHeld || game.rightHeld || game.thrustHeld || game.fireHeld || game.ship.thrust {
		t.Fatal("losing focus must release steering, thrust, and fire")
	}

	if game.state != StatePlaying {
		t.Fatal("losing board focus alone must not change the game state")
	}
}

func TestBoardBackControl(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	game := NewGame()
	board := NewBoard(game)
	backs := 0
	board.onBack = func() { backs++ }
	w := test.NewWindow(board)
	defer w.Close()

	for _, size := range []fyne.Size{fyne.NewSize(360, 800), fyne.NewSize(640, 600)} {
		w.Resize(size)
		for _, state := range []GameState{StateAttract, StatePlaying, StatePaused, StateGameOver} {
			game.state = state
			board.Refresh()
			r := board.renderer
			if r.back.Position().X+r.back.Size().Width >= r.score.Position().X {
				t.Fatal("back control must sit to the left of SCORE without overlap")
			}
			if r.score.Position().X+r.score.MinSize().Width >= (game.w-r.wave.MinSize().Width)/2 {
				t.Fatalf("score overlaps the wave at size %v", size)
			}
			before := backs
			bullets := len(game.bullets)
			test.TapCanvas(w.Canvas(), r.back.Position().Add(fyne.NewPos(5, 5)))
			if backs != before+1 || game.state != state || len(game.bullets) != bullets {
				t.Fatalf("back control must return home without activating the game in state %v", state)
			}
		}
	}
}
