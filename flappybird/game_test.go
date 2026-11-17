package flappybird

import (
	"math"
	"testing"
)

func TestFlightPauseAndRestart(t *testing.T) {
	g := NewGame()
	y := g.birdY
	if g.Tick(1) || g.birdY != y {
		t.Fatal("ready bird must wait for input")
	}
	var sounds []Sound
	g.onSound = func(s Sound) { sounds = append(sounds, s) }
	g.Flap()
	g.Tick(0.1)
	if g.state != StatePlaying || g.birdY >= y || g.vy <= flapSpeed {
		t.Fatal("flap must lift the bird while gravity slows its ascent")
	}
	g.Flap()
	if g.vy != flapSpeed || len(sounds) != 2 || sounds[0] != SoundFlap {
		t.Fatal("each flap must reset upward velocity and emit a sound")
	}
	g.Pause()
	y, x, vy := g.birdY, g.pipes[0].x, g.vy
	if g.Tick(1) || g.birdY != y || g.pipes[0].x != x || g.vy != vy {
		t.Fatal("pause must freeze flight and pipes")
	}
	g.Flap()
	if g.state != StatePlaying || g.vy != vy || len(sounds) != 2 {
		t.Fatal("resuming must not apply an unexpected flap")
	}
	g.score, g.best = 4, 9
	g.state = StateGameOver
	g.Flap()
	if g.state != StatePlaying || g.score != 0 || g.best != 9 || g.birdY != 225 || g.vy != flapSpeed {
		t.Fatal("retry must reset the round, start flight, and preserve the best score")
	}
}

func TestPipeScoringAndRecycling(t *testing.T) {
	g := NewGame()
	g.Flap()
	g.vy = 0
	g.birdY = 250
	g.pipes[0] = pipe{x: birdX - birdRadius - pipeWidth + 1, gapTop: 180}
	points := 0
	g.onSound = func(s Sound) {
		if s == SoundPoint {
			points++
		}
	}
	g.Tick(0.02)
	g.Tick(0.02)
	if g.state != StatePlaying || g.score != 1 || g.best != 1 || points != 1 {
		t.Fatal("fully clearing a safe pipe must score exactly once")
	}
	g.pipes[0].x = -pipeWidth - 1
	rightmost := g.pipes[2].x
	g.Tick(0.01)
	p := g.pipes[0]
	if math.Abs(float64(p.x-(rightmost-scrollSpeed*0.01+pipeSpacing))) > 0.01 || p.scored {
		t.Fatal("offscreen pipes must recycle after the last pipe with scoring reset")
	}
	if p.gapTop < 90 || p.gapTop+pipeGap > groundY-55 {
		t.Fatal("random gaps must leave safe margins above and below")
	}
}

func TestFatalCollisions(t *testing.T) {
	for _, tc := range []struct {
		name string
		y    float32
		p    pipe
	}{
		{"ceiling", birdRadius - 1, pipe{x: 400, gapTop: 180}},
		{"ground", groundY - birdRadius + 1, pipe{x: 400, gapTop: 180}},
		{"upper pipe", 150, pipe{x: birdX - 10, gapTop: 180}},
		{"lower pipe", 350, pipe{x: birdX - 10, gapTop: 180}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			g := NewGame()
			g.Flap()
			g.birdY, g.vy = tc.y, 0
			g.pipes[0] = tc.p
			hits := 0
			g.onSound = func(s Sound) {
				if s == SoundHit {
					hits++
				}
			}
			g.Tick(0.01)
			y := g.birdY
			g.Tick(0.1)
			if g.state != StateGameOver || hits != 1 || g.score != 0 || g.birdY != y {
				t.Fatal("a fatal collision must stop the round and emit just one hit")
			}
		})
	}
}

func TestLargeTickCannotSkipPipe(t *testing.T) {
	g := NewGame()
	g.Flap()
	g.birdY, g.vy = 100, 0
	g.pipes[0] = pipe{x: birdX + 30, gapTop: 180}
	g.Tick(1.5)
	if g.state != StateGameOver || g.score != 0 || g.pipes[0].x+pipeWidth < birdX-birdRadius {
		t.Fatal("substeps must detect a pipe collision before a long frame can skip it")
	}
}
