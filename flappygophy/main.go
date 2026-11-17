package flappygophy

import (
	"image"
	"image/color"
	"math/rand"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
	"github.com/xdung24/games/internal/navigation"
)

// ── Game constants (logical pixel space 400×600) ─────────────────────────────

const (
	gameW = 400.0
	gameH = 600.0

	birdX      = 80.0
	birdRadius = 20.0

	gravity   = 0.28
	flapForce = -6.5

	pipeW      = 65.0
	pipeGap    = 160.0
	pipeSpeed  = 2.8
	pipePeriod = 90 // frames between new pipe pairs

	groundH = 70.0
)

// ── State ─────────────────────────────────────────────────────────────────────

type State int

const (
	StateStart State = iota
	StatePlaying
	StateOver
	StatePaused
)

// ── Pipe ──────────────────────────────────────────────────────────────────────

type Pipe struct {
	x, gapY float64
}

// ── Game ──────────────────────────────────────────────────────────────────────

type Game struct {
	mu        sync.Mutex
	state     State
	birdY     float64
	birdVY    float64
	pipes     []Pipe
	score     int
	hiScore   int
	frame     int
	flapTimer int
	deathTime time.Time
	prefs     fyne.Preferences

	scoreText *canvas.Text
	msgText   *canvas.Text
	subText   *canvas.Text
}

func NewGame(prefs fyne.Preferences) *Game {
	g := &Game{
		birdY:   gameH / 2,
		prefs:   prefs,
		hiScore: prefs.IntWithFallback("flappygophy.hiScore", 0),
	}

	g.scoreText = canvas.NewText("0", color.White)
	g.scoreText.TextSize = 44
	g.scoreText.TextStyle = fyne.TextStyle{Bold: true}
	g.scoreText.Alignment = fyne.TextAlignCenter

	g.msgText = canvas.NewText("FLAPPY GOPHY", color.White)
	g.msgText.TextSize = 34
	g.msgText.TextStyle = fyne.TextStyle{Bold: true}
	g.msgText.Alignment = fyne.TextAlignCenter

	txt := "hit space/enter to play"
	if fyne.CurrentDevice().IsMobile() {
		txt = "tap to play"
	}
	g.subText = canvas.NewText(txt, color.RGBA{R: 255, G: 240, B: 80, A: 255})
	g.subText.TextSize = 18
	g.subText.Alignment = fyne.TextAlignCenter
	return g
}

// flap is called on Space key or tap – transitions game state and applies upward impulse.
func (g *Game) flap() {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch g.state {
	case StatePaused:
		g.state = StatePlaying
	case StateStart:
		g.state = StatePlaying
		g.birdVY = flapForce
		g.flapTimer = 20
		go playJingle()
	case StatePlaying:
		g.birdVY = flapForce
		g.flapTimer = 20
		go playFlap()
	case StateOver:
		if time.Since(g.deathTime) < 100*time.Millisecond {
			return
		}
		g.birdY = gameH / 2
		g.birdVY = flapForce
		g.pipes = nil
		g.score = 0
		g.frame = 0
		g.flapTimer = 20
		g.state = StatePlaying
		go playJingle()
	}
}

func (g *Game) Pause() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state == StatePlaying {
		g.state = StatePaused
	}
}

func (g *Game) TogglePause() {
	g.mu.Lock()
	defer g.mu.Unlock()
	switch g.state {
	case StatePlaying:
		g.state = StatePaused
	case StatePaused:
		g.state = StatePlaying
	}
}

// endRound is called with mu held.
func (g *Game) endRound() {
	g.state = StateOver
	g.deathTime = time.Now()
	if g.score > g.hiScore {
		g.hiScore = g.score
		g.prefs.SetInt("flappygophy.hiScore", g.hiScore)
	}
	go playSplat()
}

// update advances physics and reports whether the scene changed.
func (g *Game) update() bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.state != StatePlaying {
		return false
	}

	g.frame++
	if g.flapTimer > 0 {
		g.flapTimer--
	}
	g.birdVY += gravity
	g.birdY += g.birdVY

	// Spawn a new pipe pair.
	if g.frame%pipePeriod == 1 {
		lo := pipeGap/2 + 50
		hi := gameH - groundH - pipeGap/2 - 50
		gapY := lo + rand.Float64()*(hi-lo)
		g.pipes = append(g.pipes, Pipe{x: gameW + 10, gapY: gapY})
	}

	// Move pipes left; score when bird clears the right edge.
	keep := g.pipes[:0]
	for _, p := range g.pipes {
		p.x -= pipeSpeed
		oldRight := p.x + pipeW + pipeSpeed
		if oldRight >= birdX && p.x+pipeW < birdX {
			g.score++
		}
		if p.x+pipeW > 0 {
			keep = append(keep, p)
		}
	}
	g.pipes = keep

	// Ground / ceiling collision.
	if g.birdY-birdRadius <= 0 || g.birdY+birdRadius >= gameH-groundH {
		g.endRound()
		return true
	}
	// Pipe collision.
	for _, p := range g.pipes {
		if pipeCollides(g.birdY, p) {
			g.endRound()
			return true
		}
	}
	return true
}

func pipeCollides(birdY float64, p Pipe) bool {
	if birdX+birdRadius < p.x || birdX-birdRadius > p.x+pipeW {
		return false
	}
	topEdge := p.gapY - pipeGap/2
	botEdge := p.gapY + pipeGap/2
	return birdY-birdRadius < topEdge || birdY+birdRadius > botEdge
}

// ── Rendering ─────────────────────────────────────────────────────────────────

// groundHash returns a deterministic value in [-32, 31] for a given (x, y).
func groundHash(x, y int) int {
	h := uint32(x)*2654435761 ^ uint32(y)*2246822519
	h ^= h >> 15
	h *= 2246822519
	h ^= h >> 13
	return int(h&0x3F) - 32
}

// drawGround renders a rocky ledge with horizontal strata, per-pixel grain,
// a dark shadow lip at the top, and sparse crack pixels.
func drawGround(img *image.RGBA, gndY, w, h int) {
	for y := gndY; y < h; y++ {
		off := y * img.Stride
		dy := y - gndY
		// Strata: a single band offset shared across all x in a 10-px-tall row.
		band := groundHash(0, (y/10)*10) / 2 // [-16, 15]
		for x := 0; x < w; x++ {
			var rv, gv, bv uint8
			if dy < 4 {
				// Dark overhang shadow.
				shade := uint8(50 + dy*8)
				rv, gv, bv = shade, shade-5, shade-10
			} else {
				grain := groundHash(x, y) >> 2 // [-8, 7] per-pixel noise
				v := band + grain              // [-24, 22]
				rv = uint8(145 + v)
				gv = uint8(133 + v)
				bv = uint8(118 + v)
				// Sparse dark crack pixels (~1 in 128).
				if groundHash(x+37, y+13)&0x7F == 0 {
					rv, gv, bv = 72, 64, 56
				}
			}
			img.Pix[off+x*4] = rv
			img.Pix[off+x*4+1] = gv
			img.Pix[off+x*4+2] = bv
			img.Pix[off+x*4+3] = 255
		}
	}
}

// ── Tap widget ────────────────────────────────────────────────────────────────

// tapWidget is an invisible widget that fills its container and captures taps.
type tapWidget struct {
	widget.BaseWidget
	onTap   func()
	onPause func()
	onBack  func()
	escape  navigation.EscapeHandler
}

func newTapWidget(fn func()) *tapWidget {
	t := &tapWidget{onTap: fn}
	t.ExtendBaseWidget(t)
	return t
}

func (t *tapWidget) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (t *tapWidget) MouseDown(*desktop.MouseEvent) {
	t.activate()
}

func (t *tapWidget) activate() {
	t.escape.Reset()
	if c := fyne.CurrentApp().Driver().CanvasForObject(t); c != nil {
		c.Focus(t)
	}
	if t.onTap != nil {
		t.onTap()
	}
}

func (t *tapWidget) MouseUp(*desktop.MouseEvent) {}

func (t *tapWidget) TouchDown(*mobile.TouchEvent) {
	t.activate()
}

func (t *tapWidget) TouchUp(*mobile.TouchEvent)     {}
func (t *tapWidget) TouchCancel(*mobile.TouchEvent) {}

func (t *tapWidget) FocusGained()           {}
func (t *tapWidget) FocusLost()             { t.escape.Reset() }
func (t *tapWidget) TypedRune(rune)         { t.escape.Reset() }
func (t *tapWidget) KeyDown(*fyne.KeyEvent) {}
func (t *tapWidget) KeyUp(e *fyne.KeyEvent) { t.escape.KeyUp(e.Name) }

func (t *tapWidget) TypedKey(e *fyne.KeyEvent) {
	if t.escape.TypedKey(e.Name, t.onPause, t.onBack) {
		return
	}
	switch e.Name {
	case fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter, fyne.KeyUp:
		t.activate()
	}
}

// ── Overlay (clouds + ground bar) ─────────────────────────────────────────────

const numClouds = 5

type cloudSpec struct{ baseX, baseY, radius, speed float32 }

// cloudDefs lists each cloud.  All clouds are positioned near the top of the
// screen; negative baseY centres them above the safe-area edge so their lower
// puffs emerge into the status-bar inset area.
var cloudDefs = [numClouds]cloudSpec{
	{60, -22, 32, 10},
	{230, -10, 28, 8},
	{355, -5, 20, 13},
	{130, 18, 24, 7},
	{375, -15, 23, 12},
}
