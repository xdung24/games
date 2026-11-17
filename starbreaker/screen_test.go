package starbreaker

import (
	"fmt"
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func BenchmarkBoardRefresh(b *testing.B) {
	a := test.NewApp()
	defer a.Quit()
	for _, rocks := range []int{5, 30} {
		b.Run(fmt.Sprintf("rocks=%d", rocks), func(b *testing.B) {
			game := newPlayingGame()
			for i := 0; i < rocks; i++ {
				game.spawnAsteroid(sizeMedium, vec{100 + float32(i%5)*100, 150 + float32(i/5)*70})
			}
			for i := 0; i < 80; i++ {
				game.particles = append(game.particles, &Particle{
					pos: vec{200, 300}, vel: vec{40, 50}, life: 0.5, maxLife: 1,
				})
			}
			board := NewBoard(game)
			w := test.NewWindow(board)
			defer w.Close()
			w.Resize(fyne.NewSize(640, 600))
			board.Refresh()
			b.ReportAllocs()
			for b.Loop() {
				game.asteroids[0].angle += 0.01
				board.Refresh()
			}
		})
	}
}

func TestBoardReusesVectorsAcrossMotionAndSpawning(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := newPlayingGame()
	board := NewBoard(g)
	w := test.NewWindow(board)
	defer w.Close()
	g.ship.alive, g.lives = false, 0
	rock := &Asteroid{pos: vec{100, 100}, radius: 15, shape: shipShape}
	bullet := &Bullet{pos: vec{150, 150}, vel: vec{40, 0}}
	particle := &Particle{pos: vec{200, 200}, vel: vec{0, 40}, life: 0.5, maxLife: 1}
	g.asteroids, g.bullets, g.particles = []*Asteroid{rock}, []*Bullet{bullet}, []*Particle{particle}
	muted, reads := false, 0
	board.muted = func() bool { reads++; return muted }
	board.Refresh()
	r := board.renderer
	objects := append([]fyne.CanvasObject(nil), r.entities.Objects...)
	if len(objects) != 3 {
		t.Fatalf("got %d entities, want rock, bullet, particle", len(objects))
	}
	polygon := r.polygons[0].object
	pointBuffer := &polygon.Points[0]
	rock.pos, rock.angle = vec{120, 110}, 0.7
	bullet.pos, bullet.fromSaucer = vec{160, 170}, true
	particle.pos, particle.life = vec{220, 210}, 0.25
	g.score, g.wave, muted, reads = 120, 3, true, 0
	board.Refresh()
	if reads != 1 {
		t.Fatalf("one board refresh synchronized the scene %d times", reads)
	}
	for i, object := range objects {
		if r.entities.Objects[i] != object {
			t.Fatalf("moving entity %d replaced its canvas object", i)
		}
	}
	if &polygon.Points[0] != pointBuffer {
		t.Fatal("rotation replaced the vertex buffer")
	}
	assertPolygonShape(t, polygon, rock.shape, rock.pos, rock.radius, rock.angle)
	if r.lines[0].object.Position1 != fyne.NewPos(160, 170) ||
		r.lines[0].stroke != colEnemyShot || r.lines[0].object.StrokeColor != &r.lines[0].stroke {
		t.Fatal("reused bullet did not update its position and colour")
	}
	if r.lines[1].object.Position1 != fyne.NewPos(220, 210) ||
		r.lines[1].stroke.A != 63 || r.lines[1].object.StrokeColor != &r.lines[1].stroke {
		t.Fatal("reused particle did not update its position and fade")
	}
	if r.score.Text != "SCORE  120" || r.wave.Text != "WAVE 3" || !r.muteIcon.Visible() {
		t.Fatal("HUD changes were lost while reusing the scene")
	}
	g.asteroids, g.bullets, g.particles = nil, nil, nil
	board.Refresh()
	if len(r.entities.Objects) != 0 {
		t.Fatal("removed entities remain in the drawing tree")
	}
	rock.shape = saucerShape
	g.asteroids = []*Asteroid{rock}
	board.Refresh()
	if len(r.entities.Objects) != 1 || r.entities.Objects[0] != polygon {
		t.Fatal("spawning must reuse a free polygon without resurrecting old entities")
	}
	assertPolygonShape(t, polygon, rock.shape, rock.pos, rock.radius, rock.angle)
}

func TestBoardPreservesWrappedShipHaloAndFlame(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := newPlayingGame()
	board := NewBoard(g)
	w := test.NewWindow(board)
	defer w.Close()
	g.ship.pos, g.ship.angle = vec{1, 1}, 0.7
	g.ship.invuln, g.ship.thrust, g.lives = 0, true, 2
	board.Refresh()
	r := board.renderer
	offsets, count := g.wrapOffsets(g.ship.pos, g.shipRadius())
	if len(r.entities.Objects) != count*3+g.lives {
		t.Fatal("wrapped hulls, halos, flames or lives are missing")
	}
	flick := 0.6 + 0.4*float32(math.Sin(float64(g.ship.pos.x+g.ship.pos.y)))
	flame := []vec{{-0.4, 0.5}, {0, 1 + flick*0.6}, {0.4, 0.5}}
	for i, off := range offsets[:count] {
		for j, want := range []struct {
			shape []vec
			col   color.NRGBA
			width float32
		}{
			{shipShape, colShipGlow, r.strokeWidth() * 3},
			{shipShape, colShip, r.strokeWidth()},
			{flame, colFlame, r.strokeWidth()},
		} {
			p := r.polygons[i*3+j]
			assertPolygonShape(t, p.object, want.shape, g.ship.pos.add(off), g.shipRadius(), g.ship.angle)
			if p.stroke != want.col || p.object.StrokeWidth != want.width {
				t.Fatal("reused ship polygon lost its halo, hull or flame styling")
			}
		}
	}
	g.ship.thrust = false
	board.Refresh()
	if len(r.entities.Objects) != count*2+g.lives {
		t.Fatal("flames remain after releasing thrust")
	}
	g.ship.invuln, g.lives = 0.25, 1
	board.Refresh()
	if len(r.entities.Objects) != 1 {
		t.Fatal("spawn blink or changed life count left stale ship polygons")
	}
}

func assertPolygonShape(t *testing.T, polygon *canvas.ArbitraryPolygon, shape []vec, centre vec, radius, angle float32) {
	t.Helper()
	if len(polygon.Points) != len(shape) {
		t.Fatalf("got %d vertices, want %d", len(polygon.Points), len(shape))
	}
	for i, point := range polygon.Points {
		want := centre.add(shape[i].scale(radius).rot(angle))
		got := polygon.Position().Add(point)
		if !approx(got.X, want.x) || !approx(got.Y, want.y) {
			t.Fatalf("vertex %d = %v, want %v", i, got, want)
		}
	}
}

func TestBoardClockAndFinalFramesFollowGameLifecycle(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := newPlayingGame()
	board := NewBoard(g)
	w := test.NewWindow(board)
	defer w.Close()
	board.tick(0.02)
	r := board.renderer
	if !approx(r.bg.Uniforms["time"], 0.02) {
		t.Fatal("playing must advance the nebula with the simulation")
	}
	g.Pause()
	board.tick(0.02)
	if !r.overlay.Visible() || r.title.Text != "PAUSED" || !approx(r.bg.Uniforms["time"], 0.02) {
		t.Fatal("pausing must draw the final frame and freeze the nebula")
	}
	reads := 0
	board.muted = func() bool { reads++; return false }
	board.tick(0.02)
	if reads != 0 {
		t.Fatal("an unchanged paused scene must not synchronize")
	}
	g.state, g.dirty = StateGameOver, true
	board.tick(0.02)
	if r.title.Text != "GAME OVER" || !approx(r.bg.Uniforms["time"], 0.02) {
		t.Fatal("game over must draw once without restarting the background")
	}
	g.state, g.dirty = StateAttract, true
	board.tick(0.02)
	if !approx(r.bg.Uniforms["time"], 0.04) {
		t.Fatal("the intentional attract animation must still move")
	}
	board.Hide()
	reads = 0
	board.tick(0.02)
	if reads != 0 || !approx(board.backgroundTime, 0.04) {
		t.Fatal("hidden boards must not tick or animate")
	}
	board.Show()
	g.state, g.dirty = StatePlaying, true
	board.tick(0.02)
	if r.overlay.Visible() || r.title.Visible() || !approx(r.bg.Uniforms["time"], 0.06) {
		t.Fatal("resuming must preserve background time and remove the idle banner")
	}
}

func TestBoardEscapeReturnsHome(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	for _, state := range []GameState{StateAttract, StatePlaying, StatePaused, StateGameOver} {
		game := newPlayingGame()
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
