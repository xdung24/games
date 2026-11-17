package flappygophy

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"math"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func BenchmarkSceneRefresh(b *testing.B) {
	a := test.NewApp()
	defer a.Quit()
	for _, size := range [][2]int{{400, 600}, {1200, 1800}, {1920, 1080}} {
		b.Run(fmt.Sprintf("%dx%d", size[0], size[1]), func(b *testing.B) {
			g := NewGame(a.Preferences())
			g.state = StatePlaying
			g.pipes = []Pipe{{x: 180, gapY: 280}, {x: 350, gapY: 220}}
			scene := newGameScene(g)
			scene.Resize(fyne.NewSize(float32(size[0]), float32(size[1])))
			now := time.Unix(0, 0)
			scene.refresh(now)
			b.ReportAllocs()
			for b.Loop() {
				g.birdY = 601 - g.birdY
				g.pipes[0].x = 361 - g.pipes[0].x
				now = now.Add(time.Second / 60)
				scene.refresh(now)
			}
		})
	}
}

func TestSceneMovesSpritesWithoutResizingOrReplacingTextures(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := NewGame(a.Preferences())
	g.state = StatePlaying
	g.pipes = []Pipe{{x: 180, gapY: 280}}
	scene := newGameScene(g)
	scene.Resize(fyne.NewSize(400, 600))
	scene.refresh(time.Unix(0, 0))
	objects := append([]fyne.CanvasObject(nil), scene.objects...)
	pipeSize, pipeTexture := scene.pipes[0][0].edge.Size(), scene.pipes[0][0].fill
	gopherTexture := scene.gopher[0].Resource
	g.birdY = 420
	g.pipes[0].x = -20
	g.flapTimer = 20
	scene.refresh(time.Unix(1, 0))
	if scene.pipes[0][0].edge.Position().X != -20 || scene.pipes[0][0].edge.Size() != pipeSize ||
		scene.pipes[0][0].fill != pipeTexture {
		t.Fatal("moving offscreen must clip the pipe without resizing or replacing its texture")
	}
	if scene.gopher[0].Visible() || !scene.gopher[1].Visible() ||
		scene.gopher[1].Position() != fyne.NewPos(48, 388) {
		t.Fatal("flapping must move the gopher and select its cached raised-paw pose")
	}
	g.flapTimer = 0
	g.pipes = nil
	scene.refresh(time.Unix(2, 0))
	if !scene.gopher[0].Visible() || scene.gopher[0].Resource != gopherTexture {
		t.Fatal("the resting pose must reuse its original texture")
	}
	for _, parts := range scene.pipes {
		for _, part := range parts {
			if part.edge.Visible() || part.fill.Visible() {
				t.Fatal("removed pipes must not leave stale sprites on screen")
			}
		}
	}
	for i, object := range objects {
		if scene.objects[i] != object {
			t.Fatal("animation must reuse the existing scene objects")
		}
	}
}

func TestSceneResizeKeepsGeometryAndFullResolutionDetail(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := NewGame(a.Preferences())
	g.state = StatePlaying
	g.pipes = []Pipe{{x: 180, gapY: 280}}
	scene := newGameScene(g)
	closeTo := func(got, want float32) bool { return math.Abs(float64(got-want)) < 0.001 }
	for _, size := range []fyne.Size{fyne.NewSize(400, 600), fyne.NewSize(1200, 1800), fyne.NewSize(1920, 1080)} {
		scene.Resize(size)
		scene.refresh(time.Unix(0, 0))
		sx, sy := size.Width/gameW, size.Height/gameH
		diameter := 64 * min(sx, sy)
		pose := scene.gopher[0]
		if !closeTo(pose.Size().Width, diameter) || !closeTo(pose.Size().Height, diameter) ||
			!closeTo(pose.Position().X+diameter/2, birdX*sx) ||
			!closeTo(pose.Position().Y+diameter/2, 300*sy) {
			t.Fatalf("gopher proportions or position changed at %v", size)
		}
		top, cap := scene.pipes[0][0].edge, scene.pipes[0][1].edge
		if !closeTo(top.Size().Width, pipeW*sx) || !closeTo(top.Size().Height, 186*sy) ||
			!closeTo(cap.Position().Y+cap.Size().Height, 200*sy) ||
			!closeTo(scene.ground.Position().Y, (gameH-groundH)*sy) {
			t.Fatalf("pipe gaps and ground must match the simulation at %v", size)
		}
	}
	if bounds := scene.ground.Generator(1200, 210).Bounds(); bounds != image.Rect(0, 0, 1200, 210) {
		t.Fatalf("ground detail must retain the requested display resolution, got %v", bounds)
	}
}

func TestGopherArtworkIsCenteredAndUnclipped(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	for _, resource := range []fyne.Resource{gopherRest, gopherFlap} {
		pose := canvas.NewImageFromResource(resource)
		w := test.NewWindow(pose)
		w.SetPadded(false)
		w.Resize(fyne.NewSize(256, 256))
		pose.Refresh()
		if pose.Image == nil {
			t.Fatalf("%s did not render", resource.Name())
		}
		bounds := pose.Image.Bounds()
		center := color.NRGBAModel.Convert(pose.Image.At(bounds.Dx()/2, bounds.Dy()/2))
		if center != (color.NRGBA{R: 99, G: 136, B: 168, A: 255}) {
			t.Fatalf("%s must center the full gopher body, got %v", resource.Name(), center)
		}
		for y := 0; y < bounds.Dy(); y++ {
			for _, x := range []int{0, bounds.Dx() - 1} {
				if _, _, _, alpha := pose.Image.At(x, y).RGBA(); alpha != 0 {
					t.Fatalf("%s is clipped against an image edge", resource.Name())
				}
			}
		}
		w.Close()
	}
}

func TestUpdateRequestsRenderingOnlyWhenPlaying(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	for _, state := range []State{StateStart, StatePaused, StateOver} {
		g := NewGame(a.Preferences())
		g.state = state
		if g.update() || g.frame != 0 || g.birdY != gameH/2 {
			t.Fatalf("idle state %v must not advance or request a redraw", state)
		}
	}
	g := NewGame(a.Preferences())
	g.state = StatePlaying
	if !g.update() || g.frame != 1 {
		t.Fatal("playing must advance physics and request a redraw")
	}
	g.birdY = gameH - groundH
	if !g.update() || g.state != StateOver {
		t.Fatal("the collision frame must redraw the final position and game-over state")
	}
	if g.update() {
		t.Fatal("game-over must stop continuous rendering after the final frame")
	}
}

func TestScreenNavigationPreservesRound(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := test.NewWindow(nil)
	defer w.Close()
	g := NewGame(a.Preferences())
	backs := 0
	s := newScreen(w, g, func() { backs++ })
	s.Show()
	if w.Content() != s.content || w.Canvas().Focused() != s.input {
		t.Fatal("show must install the game and focus its input")
	}
	s.input.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	g.update()
	g.score = 3
	y, vy, frame := g.birdY, g.birdVY, g.frame
	s.Hide()
	g.update()
	if s.content.Visible() || g.state != StatePaused || g.birdY != y || g.birdVY != vy || g.frame != frame {
		t.Fatal("hidden games must remain frozen")
	}
	s.Show()
	if !s.content.Visible() || g.state != StatePaused || g.score != 3 || g.msgText.Text != "PAUSED" {
		t.Fatal("returning must preserve paused progress and show the pause message")
	}
	s.input.MouseDown(&desktop.MouseEvent{})
	if g.state != StatePlaying || g.birdVY != vy {
		t.Fatal("a tap must resume without applying a flap")
	}
	s.Pause()
	if g.state != StatePaused {
		t.Fatal("leaving the foreground must pause play")
	}
	test.Tap(s.pause)
	if g.state != StatePlaying || w.Canvas().Focused() != s.input {
		t.Fatal("pause control must resume and restore game focus")
	}
	escape := &fyne.KeyEvent{Name: fyne.KeyEscape}
	s.input.TypedKey(escape)
	s.input.TypedKey(escape)
	if g.state != StatePaused || backs != 0 {
		t.Fatal("Esc and held repeats must pause without returning Home")
	}
	s.input.KeyUp(escape)
	s.input.TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
	s.input.TypedKey(escape)
	if backs != 0 {
		t.Fatal("another key must cancel the two-Esc sequence")
	}
	s.input.KeyUp(escape)
	s.input.TypedKey(escape)
	if backs != 1 || s.content.Visible() || g.score != 3 {
		t.Fatal("second separate Esc must return Home and preserve progress")
	}
	s.Show()
	s.input.TypedKey(escape)
	if backs != 1 {
		t.Fatal("showing a hidden game must start a fresh Esc sequence")
	}
	test.Tap(s.back)
	if backs != 2 || s.content.Visible() {
		t.Fatal("Home control must hide the game")
	}
}

func TestBestScoreSavedBeforeLeavingGameOver(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	prefs := a.Preferences()
	prefs.SetInt("hiScore", 99)
	g := NewGame(prefs)
	g.state, g.score, g.birdY = StatePlaying, 5, gameH-groundH
	g.update()
	if g.state != StateOver || g.hiScore != 5 || prefs.Int("flappygophy.hiScore") != 5 || prefs.Int("hiScore") != 99 {
		t.Fatal("game over must save a game-specific best score without waiting for retry")
	}
	loaded := NewGame(prefs)
	if loaded.hiScore != 5 {
		t.Fatal("a new session must load the saved best")
	}
	g.deathTime = time.Now().Add(-time.Second)
	g.flap()
	if g.state != StatePlaying || g.score != 0 || g.hiScore != 5 || len(g.pipes) != 0 || g.birdVY != flapForce {
		t.Fatal("retry must retain the saved best and reset the round")
	}
}

func TestSharedAudioPCMFormat(t *testing.T) {
	samples := []float64{-2, -0.5, 0, 0.5, 2}
	pcm := genSamples(len(samples), func(i int) float64 { return samples[i] })
	want := []int16{-32767, -16383, 0, 16383, 32767}
	if len(pcm) != len(want)*2 {
		t.Fatal("shared device requires one 16-bit mono sample per frame")
	}
	for i, value := range want {
		if got := int16(binary.LittleEndian.Uint16(pcm[i*2:])); got != value {
			t.Fatalf("sample %d: got %d, want %d", i, got, value)
		}
	}
	if len(silenceBuf(time.Second)) != audioRate*2 {
		t.Fatal("silence must use the same mono sample format and sample rate")
	}
}
