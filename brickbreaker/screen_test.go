package brickbreaker

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
	"github.com/xdung24/games/internal/touchcontrols"
)

func TestTouchControlsFloatAbovePaddle(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	b := NewBoard(newTestGame())
	s := &Screen{board: b}
	panel := s.newControls(b)
	w := test.NewWindow(panel)
	defer w.Close()
	for _, size := range []fyne.Size{fyne.NewSize(320, 568), fyne.NewSize(430, 740), fyne.NewSize(932, 430)} {
		panel.Resize(size)
		if b.Size() != size {
			t.Fatalf("overlay controls must not shrink the playfield at %v", size)
		}
		for _, button := range []*touchcontrols.Button{panel.Left[0], panel.Right[0]} {
			if button.Position().Y < 0 ||
				button.Position().Y+button.Size().Height >= b.game.paddleY {
				t.Fatalf("%s must float inside the playfield above the paddle at %v", button.Name, size)
			}
		}
	}
}

func TestBallPassesThroughTouchControls(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := newTestGame()
	b := NewBoard(g)
	s := &Screen{board: b}
	panel := s.newControls(b)
	w := test.NewWindow(panel)
	defer w.Close()
	for _, size := range []fyne.Size{fyne.NewSize(320, 568), fyne.NewSize(430, 740), fyne.NewSize(932, 430)} {
		panel.Resize(size)
		for _, button := range []*touchcontrols.Button{panel.Left[0], panel.Right[0]} {
			g.state = StatePlaying
			g.ballX = button.Position().X + button.Size().Width/2
			g.ballY = button.Position().Y - 2*g.ballR
			g.ballVX, g.ballVY = 0, (button.Size().Height+4*g.ballR)/0.2
			x, velocity := g.ballX, g.ballVY
			wantY := button.Position().Y + button.Size().Height + 2*g.ballR
			for i := 0; i < 12; i++ {
				g.Tick(1.0 / 60)
			}
			if g.state != StatePlaying || g.ballX != x || g.ballVY != velocity ||
				math.Abs(float64(g.ballY-wantY)) > 0.01 {
				t.Fatalf("ball must pass through %s without bouncing or pausing at %v", button.Name, size)
			}
		}
	}
}

func TestGoTouchControlsMovePaddleAndPreserveKeyboardHolds(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := newTestGame()
	b := NewBoard(g)
	w := test.NewWindow(b)
	defer w.Close()
	s := &Screen{window: w, game: g, board: b}
	s.controls = s.newControls(b)
	s.content = s.controls
	b.onFocusLost = s.controls.Reset
	s.Show()
	if len(s.controls.Left) != 1 || len(s.controls.Right) != 1 || len(s.controls.System) != 0 {
		t.Fatal("Brick Breaker needs exactly one left and one right control")
	}
	left, right := s.controls.Left[0], s.controls.Right[0]
	start := g.paddleX
	left.TouchDown(&mobile.TouchEvent{ID: 1})
	if g.state != StateReady {
		t.Fatal("pressing a control must not launch the ball underneath")
	}
	g.Tick(0.05)
	if g.paddleX >= start {
		t.Fatal("Go touch press did not move the paddle left")
	}
	left.TouchUp(&mobile.TouchEvent{ID: 1})
	stopped := g.paddleX
	g.Tick(0.05)
	if g.paddleX != stopped {
		t.Fatal("releasing the Go control must stop the paddle")
	}
	b.KeyDown(&fyne.KeyEvent{Name: fyne.KeyRight})
	right.TouchDown(&mobile.TouchEvent{ID: 2})
	right.TouchCancel(&mobile.TouchEvent{ID: 2})
	if !g.rightHeld {
		t.Fatal("releasing touch must preserve a held physical key")
	}
	b.KeyUp(&fyne.KeyEvent{Name: fyne.KeyRight})
	left.TouchDown(&mobile.TouchEvent{ID: 3})
	right.TouchDown(&mobile.TouchEvent{ID: 4})
	s.Hide()
	if g.leftHeld || g.rightHeld {
		t.Fatal("leaving the game must clear held Go controls")
	}
	s.Show()
	right.TouchDown(&mobile.TouchEvent{ID: 4})
	g.Tick(0.05)
	if !g.rightHeld {
		t.Fatal("returning must accept a fresh touch, even with a reused ID")
	}
	right.TouchUp(&mobile.TouchEvent{ID: 4})
}

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
