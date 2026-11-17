package flappygophy

import (
	"encoding/binary"
	"fmt"
	"image"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func BenchmarkDrawFrame(b *testing.B) {
	a := test.NewApp()
	defer a.Quit()
	for _, size := range [][2]int{{400, 600}, {1200, 1800}, {1920, 1080}} {
		for _, web := range []bool{false, true} {
			b.Run(fmt.Sprintf("%dx%d/web=%t", size[0], size[1], web), func(b *testing.B) {
				g := NewGame(a.Preferences())
				g.state = StatePlaying
				g.pipes = []Pipe{{x: 180, gapY: 280}, {x: 350, gapY: 220}}
				draw := g.drawFrame
				if web {
					draw = g.drawWebFrame
				}
				draw(size[0], size[1])
				b.ReportAllocs()
				b.ResetTimer()
				for b.Loop() {
					draw(size[0], size[1])
				}
			})
		}
	}
}

func TestWebRasterResolution(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := NewGame(a.Preferences())
	for _, size := range []struct {
		w, h, wantW, wantH int
	}{
		{400, 600, 400, 600},
		{1200, 1800, 400, 600},
		{1170, 2532, 277, 600},
		{1920, 1080, 400, 225},
		{200, 300, 200, 300},
	} {
		got := g.drawWebFrame(size.w, size.h).Bounds()
		if got != image.Rect(0, 0, size.wantW, size.wantH) {
			t.Errorf("web frame %dx%d: got %v, want %dx%d", size.w, size.h, got, size.wantW, size.wantH)
		}
	}
	if got := g.drawFrame(800, 600).Bounds(); got != image.Rect(0, 0, 800, 600) {
		t.Fatalf("native rendering must retain full resolution, got %v", got)
	}
}

func TestDrawFrameReusesBufferWithoutTrails(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	g := NewGame(a.Preferences())
	g.state = StatePlaying
	g.pipes = []Pipe{{x: 180, gapY: 280}}
	first := g.drawFrame(200, 300)
	frame, ok := first.(*image.RGBA)
	if !ok {
		t.Fatal("frames must be RGBA for direct upload without a Fyne image conversion")
	}
	for i := 3; i < len(frame.Pix); i += 4 {
		if frame.Pix[i] != 255 {
			t.Fatal("directly painted RGBA frames must remain opaque")
		}
	}
	background := g.background
	for _, point := range []image.Point{{40, 150}, {95, 80}} {
		if first.At(point.X, point.Y) == background.At(point.X, point.Y) {
			t.Fatalf("expected a bird or pipe at %v before moving", point)
		}
	}
	g.birdY = 420
	g.pipes = nil
	second := g.drawFrame(200, 300)
	if first != second || g.background != background {
		t.Fatal("unchanged dimensions must reuse the frame and static background")
	}
	for _, point := range []image.Point{{40, 150}, {95, 80}} {
		if second.At(point.X, point.Y) != background.At(point.X, point.Y) {
			t.Fatalf("moving objects must not leave a trail at %v", point)
		}
	}
	if second.At(40, 210) == background.At(40, 210) {
		t.Fatal("the bird must appear at its new position")
	}
	g.drawFrame(300, 200)
	if g.background == background || g.frameImage.Bounds() != image.Rect(0, 0, 300, 200) {
		t.Fatal("resizing must rebuild the cached images")
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
