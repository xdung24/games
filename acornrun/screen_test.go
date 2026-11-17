package acornrun

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestEscapePausesThenReturnsHome(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	game := NewGame()
	game.Start()
	board := NewBoard(game)
	window := test.NewWindow(board)
	defer window.Close()
	returns := 0
	board.onBack = func() { returns++ }

	escape := &fyne.KeyEvent{Name: fyne.KeyEscape}
	board.TypedKey(escape)
	if game.state != StatePaused || returns != 0 {
		t.Fatal("first Escape should pause without returning Home")
	}
	board.KeyUp(escape)
	board.TypedKey(escape)
	if returns != 1 {
		t.Fatal("second Escape should return Home")
	}
}

func TestKeyboardMovementAndJumpInput(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()

	game := NewGame()
	board := NewBoard(game)
	board.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if game.state != StatePlaying {
		t.Fatal("Space should start the game from its title screen")
	}
	board.KeyUp(&fyne.KeyEvent{Name: fyne.KeySpace})
	board.KeyDown(&fyne.KeyEvent{Name: fyne.KeyRight})
	if !board.input.Right {
		t.Fatal("right key should hold rightward movement")
	}
	startX := game.player.x
	game.Update(1.0/60, board.frameInput())
	if game.player.x <= startX {
		t.Fatal("holding right should move the player")
	}
	board.KeyUp(&fyne.KeyEvent{Name: fyne.KeyRight})
	if board.input.Right {
		t.Fatal("releasing right should stop rightward movement")
	}
	board.KeyDown(&fyne.KeyEvent{Name: fyne.KeyX})
	if !board.input.Jump {
		t.Fatal("X should hold the jump input")
	}
	board.KeyUp(&fyne.KeyEvent{Name: fyne.KeyX})
	if board.input.Jump {
		t.Fatal("releasing X should stop the jump input")
	}
}
