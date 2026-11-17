package flappychick

import (
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestBoardControlsAndScreenLifecycle(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := NewGame()
	b := NewBoard(g)
	w := test.NewWindow(b)
	defer w.Close()
	s := &Screen{window: w, game: g, board: b}
	backs := 0
	b.onBack = func() {
		backs++
		s.Hide()
	}
	s.Show()
	b.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	g.score = 3
	s.Hide()
	if g.state != StatePaused || b.Visible() {
		t.Fatal("hiding must pause and hide the board")
	}
	s.Show()
	if g.state != StatePaused || g.score != 3 || w.Canvas().Focused() != b {
		t.Fatal("returning must preserve progress and restore keyboard focus")
	}
	test.Tap(b)
	if g.state != StatePlaying {
		t.Fatal("a tap must resume the round")
	}
	s.Pause()
	if g.state != StatePaused {
		t.Fatal("losing foreground must pause the round")
	}
	b.TypedKey(&fyne.KeyEvent{Name: fyne.KeyR})
	if g.state != StateReady || g.score != 0 {
		t.Fatal("restart must reset to ready")
	}
	b.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	if g.state != StatePlaying || g.vy != flapSpeed {
		t.Fatal("Up must flap for browser keyboard support")
	}
	escape := &fyne.KeyEvent{Name: fyne.KeyEscape}
	b.TypedKey(escape)
	b.TypedKey(escape)
	if g.state != StatePaused || backs != 0 {
		t.Fatal("first Esc and held repeats must pause without leaving")
	}
	b.KeyUp(escape)
	b.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	b.TypedKey(escape)
	if backs != 0 {
		t.Fatal("other input must cancel the two-Esc sequence")
	}
	b.KeyUp(escape)
	b.TypedKey(escape)
	if backs != 1 || b.Visible() {
		t.Fatal("second separate Esc must return Home")
	}
	s.Show()
	b.TypedKey(escape)
	if backs != 1 {
		t.Fatal("reopening must reset the Esc sequence")
	}
}

func TestBoardResponsiveLayoutAndButtons(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := NewGame()
	b := NewBoard(g)
	w := test.NewWindow(b)
	defer w.Close()
	backs, mutes := 0, 0
	b.onBack = func() { backs++ }
	b.onMute = func() { mutes++ }

	for _, size := range []fyne.Size{fyne.NewSize(320, 480), fyne.NewSize(640, 600), fyne.NewSize(400, 900)} {
		w.Resize(size)
		b.Refresh()
		r := b.renderer
		if math.Abs(float64(r.sky.Size().Width/r.sky.Size().Height-worldWidth/worldHeight)) > 0.001 {
			t.Fatal("resizing must preserve playfield aspect ratio")
		}
		for _, text := range []*canvas.Text{r.title, r.hint, r.detail} {
			if text.Position().X < r.ox || text.Position().X+text.MinSize().Width > r.ox+worldWidth*r.scale {
				t.Fatalf("overlay text must fit the playfield at %v", size)
			}
		}
		buttons := []*widget.Button{r.back, r.pause, r.restart, r.mute}
		for i, button := range buttons {
			if button.Position().X < r.ox ||
				button.Position().X+button.Size().Width > r.ox+worldWidth*r.scale ||
				button.Position().Y+button.Size().Height > r.oy+worldHeight*r.scale {
				t.Fatalf("controls must fit at %v", size)
			}
			if i > 0 && buttons[i-1].Position().X+buttons[i-1].Size().Width > button.Position().X {
				t.Fatal("controls must not overlap")
			}
		}
	}
	r := b.renderer
	test.Tap(r.back)
	test.Tap(r.mute)
	if backs != 1 || mutes != 1 || g.state != StateReady {
		t.Fatal("Home and mute must not flap")
	}
	test.Tap(b)
	test.Tap(r.pause)
	if g.state != StatePaused {
		t.Fatal("pause button must pause")
	}
	test.Tap(r.pause)
	if g.state != StatePlaying || w.Canvas().Focused() != b {
		t.Fatal("pause button must resume and restore board focus")
	}
	test.Tap(r.restart)
	if g.state != StateReady {
		t.Fatal("restart button must reset the round")
	}
}
