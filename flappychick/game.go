package flappychick

import "math/rand"

type State int

const (
	StateReady State = iota
	StatePlaying
	StatePaused
	StateGameOver
)

const (
	worldWidth  float32 = 360
	worldHeight float32 = 540
	groundY     float32 = 475
	birdX       float32 = 100
	birdRadius  float32 = 12
	pipeWidth   float32 = 58
	pipeGap     float32 = 145
	pipeSpacing float32 = 220
	scrollSpeed float32 = 150
	gravity     float32 = 900
	flapSpeed   float32 = -300
)

type Sound int

const (
	SoundFlap Sound = iota
	SoundPoint
	SoundHit
)

type pipe struct {
	x, gapTop float32
	scored    bool
}

// Game uses a fixed logical playfield so resizing never changes difficulty.
type Game struct {
	state       State
	birdY, vy   float32
	pipes       [3]pipe
	score, best int
	travel      float32
	onSound     func(Sound)
}

func NewGame() *Game {
	g := &Game{}
	g.Restart()
	return g
}

func (g *Game) Restart() {
	g.state = StateReady
	g.birdY, g.vy = 225, 0
	g.score, g.travel = 0, 0
	for i := range g.pipes {
		g.pipes[i] = pipe{x: worldWidth + 40 + float32(i)*pipeSpacing, gapTop: randomGap()}
	}
}

func randomGap() float32 {
	return 90 + rand.Float32()*(groundY-pipeGap-145)
}

func (g *Game) play(sound Sound) {
	if g.onSound != nil {
		g.onSound(sound)
	}
}

// Flap starts a round, resumes a paused round, or gives an upward impulse.
func (g *Game) Flap() {
	switch g.state {
	case StateGameOver:
		g.Restart()
	case StatePaused:
		g.state = StatePlaying
		return
	}
	g.state = StatePlaying
	g.vy = flapSpeed
	g.play(SoundFlap)
}

func (g *Game) Pause() {
	if g.state == StatePlaying {
		g.state = StatePaused
	}
}

func (g *Game) TogglePause() {
	if g.state == StatePaused {
		g.state = StatePlaying
	} else {
		g.Pause()
	}
}

// Tick substeps flight and pipe movement together to avoid tunnelling.
func (g *Game) Tick(dt float32) bool {
	if g.state != StatePlaying || dt <= 0 {
		return false
	}
	for dt > 0 && g.state == StatePlaying {
		step := min(dt, 1.0/120)
		dt -= step
		g.birdY += g.vy*step + gravity*step*step/2
		g.vy += gravity * step
		g.travel += scrollSpeed * step
		for i := range g.pipes {
			g.pipes[i].x -= scrollSpeed * step
		}
		if g.birdY-birdRadius <= 0 || g.birdY+birdRadius >= groundY {
			g.die()
			break
		}
		for i := range g.pipes {
			p := &g.pipes[i]
			if circleHitsRect(birdX, g.birdY, birdRadius, p.x, 0, pipeWidth, p.gapTop) ||
				circleHitsRect(birdX, g.birdY, birdRadius, p.x, p.gapTop+pipeGap, pipeWidth, groundY-p.gapTop-pipeGap) {
				g.die()
				break
			}
			if !p.scored && p.x+pipeWidth < birdX-birdRadius {
				p.scored = true
				g.score++
				g.best = max(g.best, g.score)
				g.play(SoundPoint)
			}
		}
		if g.state != StatePlaying {
			break
		}
		for i := range g.pipes {
			if g.pipes[i].x+pipeWidth < 0 {
				rightmost := g.pipes[0].x
				for _, p := range g.pipes[1:] {
					rightmost = max(rightmost, p.x)
				}
				g.pipes[i] = pipe{x: rightmost + pipeSpacing, gapTop: randomGap()}
			}
		}
	}
	return true
}

func (g *Game) die() {
	g.state = StateGameOver
	g.play(SoundHit)
}

func circleHitsRect(cx, cy, radius, x, y, w, h float32) bool {
	dx := cx - max(x, min(cx, x+w))
	dy := cy - max(y, min(cy, y+h))
	return dx*dx+dy*dy <= radius*radius
}
