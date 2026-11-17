package flappygophy

import (
	"encoding/binary"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

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
