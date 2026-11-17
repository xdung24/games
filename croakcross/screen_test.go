package croakcross

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
)

func TestEscapePausesThenReturnsHome(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

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
		t.Fatal("first Escape should pause without returning home")
	}
	board.KeyUp(escape)
	board.TypedKey(escape)
	if returns != 1 {
		t.Fatal("second Escape should return home")
	}
}
