package starbreaker

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/xdung24/games/internal/navigation"
)

// Neon palette. Entities are bright, saturated strokes on the dark shader sky;
// the glow reads from the colour against the backdrop rather than a blur pass.
var (
	colShip      = color.NRGBA{0x5E, 0xF7, 0xFF, 0xFF} // electric cyan
	colShipGlow  = color.NRGBA{0x5E, 0xF7, 0xFF, 0x44} // faint wide halo behind the hull
	colFlame     = color.NRGBA{0xFF, 0xC4, 0x4D, 0xFF} // amber thruster
	colRock      = color.NRGBA{0xC7, 0xB8, 0xFF, 0xFF} // pale violet-white
	colBullet    = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF} // white tracer
	colEnemyShot = color.NRGBA{0xFF, 0x5C, 0x8A, 0xFF} // hot pink saucer fire
	colSaucer    = color.NRGBA{0x6E, 0xFF, 0x9B, 0xFF} // toxic green
	colSpark     = color.NRGBA{0xFF, 0x9D, 0x4D, 0xFF} // explosion debris

	colText    = color.NRGBA{0xE6, 0xF6, 0xFF, 0xFF}
	colDim     = color.NRGBA{0x9A, 0xB0, 0xC8, 0xFF}
	colOverlay = color.NRGBA{0x05, 0x03, 0x10, 0xB4} // veil for pause / game-over
)

// shipShape is the ship hull as unit-radius vertices, nose up (+y is down), in
// clockwise order: nose, right wing, tail notch, left wing.
var shipShape = []vec{{0, -1}, {0.66, 0.82}, {0, 0.45}, {-0.66, 0.82}}

// saucerShape is the classic flying-saucer silhouette as unit-radius vertices.
var saucerShape = []vec{
	{-1.0, 0.05}, {-0.42, -0.18}, {-0.22, -0.18}, {-0.16, -0.5},
	{0.16, -0.5}, {0.22, -0.18}, {0.42, -0.18}, {1.0, 0.05},
	{0.5, 0.35}, {-0.5, 0.35},
}

// Board is the interactive play surface: a focusable, hoverable widget that
// renders the game and feeds player input into the model.
type Board struct {
	widget.BaseWidget
	game     *Game
	renderer *boardRenderer

	toggleMute func() bool
	muted      func() bool
	onBack     func()
	escape     navigation.EscapeHandler
}

func NewBoard(g *Game) *Board {
	b := &Board{game: g}
	b.ExtendBaseWidget(b)
	return b
}

func (b *Board) CreateRenderer() fyne.WidgetRenderer {
	r := &boardRenderer{board: b, game: b.game}

	r.bg, r.bgAnim = newNeonBackground()
	r.entities = canvasContainer()

	r.overlay = canvas.NewRectangle(colOverlay)
	r.overlay.Hide()

	r.score = newText(colText, fyne.TextAlignLeading, true)
	r.wave = newText(colDim, fyne.TextAlignCenter, true)
	r.muteIcon = canvas.NewImageFromResource(theme.VolumeMuteIcon())
	r.muteIcon.FillMode = canvas.ImageFillContain
	r.muteIcon.Hide()
	r.title = newText(colText, fyne.TextAlignCenter, true)
	r.subtitle = []*canvas.Text{
		newText(colDim, fyne.TextAlignCenter, false),
		newText(colDim, fyne.TextAlignCenter, false),
	}
	r.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if b.onBack != nil {
			b.onBack()
		}
	})
	r.back.Importance = widget.LowImportance

	b.renderer = r
	r.bgAnim.Start()
	return r
}

// Refresh redraws the whole board from the model. The game loop calls it each
// active frame.
func (b *Board) Refresh() {
	b.BaseWidget.Refresh()
	if b.renderer != nil {
		b.renderer.sync()
	}
}

// --- input: pointer ---------------------------------------------------------

func (b *Board) Tapped(*fyne.PointEvent) {
	b.escape.Reset()
	b.game.Activate()
}

// --- input: keyboard --------------------------------------------------------

func (b *Board) FocusGained() {}
func (b *Board) FocusLost() {
	b.escape.Reset()
	b.game.leftHeld = false
	b.game.rightHeld = false
	b.game.thrustHeld = false
	b.game.fireHeld = false
	b.game.ship.thrust = false
}
func (b *Board) TypedRune(rune) { b.escape.Reset() }

func (b *Board) TypedKey(e *fyne.KeyEvent) {
	if b.escape.TypedKey(e.Name, func() {
		b.game.Pause()
		b.Refresh()
	}, b.onBack) {
		return
	}
	switch e.Name {
	case fyne.KeyP:
		b.game.TogglePause()
	case fyne.KeyR:
		b.game.StartGame()
	case fyne.KeyM:
		if b.toggleMute != nil {
			b.toggleMute()
			b.Refresh() // update the HUD indicator immediately
		}
	}
}

func (b *Board) KeyDown(e *fyne.KeyEvent) {
	switch e.Name {
	case fyne.KeyLeft, fyne.KeyA:
		b.game.leftHeld = true
	case fyne.KeyRight, fyne.KeyD:
		b.game.rightHeld = true
	case fyne.KeyUp, fyne.KeyW:
		b.game.thrustHeld = true
	case fyne.KeyDown, fyne.KeyH:
		b.game.Hyperspace()
	case fyne.KeySpace:
		// Space starts a game from the menus, and fires (auto-repeat) in play.
		switch b.game.state {
		case StatePlaying:
			b.game.fireHeld = true
			b.game.Fire()
		default:
			b.game.Activate()
		}
	}
}

func (b *Board) KeyUp(e *fyne.KeyEvent) {
	b.escape.KeyUp(e.Name)
	switch e.Name {
	case fyne.KeyLeft, fyne.KeyA:
		b.game.leftHeld = false
	case fyne.KeyRight, fyne.KeyD:
		b.game.rightHeld = false
	case fyne.KeyUp, fyne.KeyW:
		b.game.thrustHeld = false
	case fyne.KeySpace:
		b.game.fireHeld = false
	}
}

// boardRenderer maps game state onto canvas objects every frame. The background
// shader, HUD and banner are persistent; the entity layer is rebuilt each frame.
type boardRenderer struct {
	board *Board
	game  *Game

	bg       *canvas.Shader
	bgAnim   *fyne.Animation
	entities *fyne.Container // absolutely-positioned vector objects, refilled per frame

	overlay  *canvas.Rectangle
	score    *canvas.Text
	wave     *canvas.Text
	muteIcon *canvas.Image // theme volume icon, swapped on mute state
	title    *canvas.Text
	subtitle []*canvas.Text // one entry per rendered line (canvas.Text is single-line)
	back     *widget.Button
}

func (r *boardRenderer) Layout(size fyne.Size) {
	r.game.Resize(size.Width, size.Height)
	r.bg.Resize(size)
	r.entities.Resize(size)
	r.overlay.Resize(size)
	r.sync()
}

func (r *boardRenderer) MinSize() fyne.Size { return fyne.NewSize(360, 360) }

func (r *boardRenderer) Destroy() {
	if r.bgAnim != nil {
		r.bgAnim.Stop()
	}
}

func (r *boardRenderer) Objects() []fyne.CanvasObject {
	objs := []fyne.CanvasObject{r.bg, r.entities, r.score, r.wave, r.muteIcon, r.overlay, r.title}
	for _, t := range r.subtitle {
		objs = append(objs, t)
	}
	return append(objs, r.back)
}

func (r *boardRenderer) Refresh() { r.sync() }

// sync rebuilds the entity layer and restyles the HUD/banner from the model.
func (r *boardRenderer) sync() {
	objs := make([]fyne.CanvasObject, 0, 64)
	objs = r.appendAsteroids(objs)
	objs = r.appendBullets(objs)
	objs = r.appendParticles(objs)
	objs = r.appendSaucer(objs)
	objs = r.appendShip(objs)
	objs = r.appendLives(objs)
	r.entities.Objects = objs
	r.entities.Refresh()

	r.refreshHUD()
	r.refreshBanner()
	canvas.Refresh(r.bg)
}

// strokeWidth scales the neon line weight to the field size.
func (r *boardRenderer) strokeWidth() float32 {
	return float32(math.Max(1.6, float64(r.game.h*0.0045)))
}

// appendShip draws the hull (with a halo), wrapped at the edges, plus the
// thruster flame. It blinks during spawn invulnerability and vanishes while dead.
func (r *boardRenderer) appendShip(objs []fyne.CanvasObject) []fyne.CanvasObject {
	g := r.game
	if g.state != StatePlaying || !g.ship.alive {
		return objs
	}
	// Blink roughly 6× a second while protected so the player reads "invincible".
	if g.ship.invuln > 0 && int(g.ship.invuln*8)%2 == 0 {
		return objs
	}
	rad := g.shipRadius()
	hull := transform(shipShape, g.ship.pos, rad, g.ship.angle)
	sw := r.strokeWidth()
	for _, off := range g.wrapOffsets(g.ship.pos, rad) {
		pts := shift(hull, off)
		objs = append(objs, strokePoly(pts, colShipGlow, sw*3)) // halo
		objs = append(objs, strokePoly(pts, colShip, sw))       // bright hull
		if g.ship.thrust {
			objs = append(objs, r.flame(off))
		}
	}
	return objs
}

// flame builds a short flickering thruster triangle behind the ship.
func (r *boardRenderer) flame(off vec) fyne.CanvasObject {
	g := r.game
	rad := g.shipRadius()
	flick := 0.6 + 0.4*float32(math.Sin(float64(g.ship.pos.x+g.ship.pos.y)))
	shape := []vec{{-0.4, 0.5}, {0, 1.0 + flick*0.6}, {0.4, 0.5}}
	pts := shift(transform(shape, g.ship.pos, rad, g.ship.angle), off)
	return strokePoly(pts, colFlame, r.strokeWidth())
}

// appendAsteroids draws every rock outline, wrapped at the edges.
func (r *boardRenderer) appendAsteroids(objs []fyne.CanvasObject) []fyne.CanvasObject {
	g := r.game
	sw := r.strokeWidth()
	for _, a := range g.asteroids {
		base := transform(a.shape, a.pos, a.radius, a.angle)
		for _, off := range g.wrapOffsets(a.pos, a.radius) {
			objs = append(objs, strokePoly(shift(base, off), colRock, sw))
		}
	}
	return objs
}

// appendSaucer draws the saucer outline (wrapped on the vertical axis).
func (r *boardRenderer) appendSaucer(objs []fyne.CanvasObject) []fyne.CanvasObject {
	g := r.game
	if !g.saucer.alive {
		return objs
	}
	rad := g.saucerRadius(g.saucer.big)
	base := transform(saucerShape, g.saucer.pos, rad, 0)
	for _, off := range g.wrapOffsets(g.saucer.pos, rad) {
		objs = append(objs, strokePoly(shift(base, off), colSaucer, r.strokeWidth()))
	}
	return objs
}

// appendBullets draws each shot as a short neon tracer along its velocity.
func (r *boardRenderer) appendBullets(objs []fyne.CanvasObject) []fyne.CanvasObject {
	g := r.game
	sw := r.strokeWidth()
	tail := g.h * 0.018
	for _, b := range g.bullets {
		dir := b.vel
		if l := dir.len(); l > 0 {
			dir = dir.scale(1 / l)
		}
		col := colBullet
		if b.fromSaucer {
			col = colEnemyShot
		}
		end := b.pos.sub(dir.scale(tail))
		objs = append(objs, line(b.pos, end, col, sw*1.1))
	}
	return objs
}

// appendParticles draws explosion/thrust debris as fading line fragments.
func (r *boardRenderer) appendParticles(objs []fyne.CanvasObject) []fyne.CanvasObject {
	g := r.game
	sw := r.strokeWidth()
	seg := g.h * 0.012
	for _, p := range g.particles {
		dir := p.vel
		if l := dir.len(); l > 0 {
			dir = dir.scale(1 / l)
		}
		a := uint8(clampF(p.life/p.maxLife, 0, 1) * 255)
		c := colSpark
		c.A = a
		objs = append(objs, line(p.pos, p.pos.sub(dir.scale(seg)), c, sw*0.8))
	}
	return objs
}

// appendLives draws the remaining-ship icons in the top-right corner.
func (r *boardRenderer) appendLives(objs []fyne.CanvasObject) []fyne.CanvasObject {
	g := r.game
	if g.state == StateAttract {
		return objs
	}
	rad := g.h * 0.018
	y := g.h * 0.045
	for i := 0; i < g.lives; i++ {
		c := vec{g.w - rad*1.6 - float32(i)*rad*2.6, y}
		objs = append(objs, strokePoly(transform(shipShape, c, rad, 0), colShip, r.strokeWidth()))
	}
	return objs
}

// refreshHUD positions and fills the score and wave readouts.
func (r *boardRenderer) refreshHUD() {
	g := r.game
	hud := min(g.h*0.032, g.w*0.035)
	if g.state == StateAttract {
		r.score.Text, r.wave.Text = "", ""
	} else {
		r.score.Text = fmt.Sprintf("SCORE  %d", g.score)
		r.wave.Text = fmt.Sprintf("WAVE %d", g.wave)
	}
	r.score.TextSize = hud
	backSize := r.back.MinSize()
	r.back.Resize(backSize)
	r.back.Move(fyne.NewPos(g.w*0.015, max(0, g.h*0.035+(hud*1.4-backSize.Height)/2)))
	r.score.Move(fyne.NewPos(r.back.Position().X+backSize.Width+theme.Padding(), g.h*0.035))

	r.wave.TextSize = hud
	r.wave.Resize(fyne.NewSize(g.w, hud*1.4))
	r.wave.Move(fyne.NewPos(0, g.h*0.035))

	// A bottom-right glyph appears only while sound is muted (toggled with M).
	if r.board.muted != nil && r.board.muted() {
		icon := g.h * 0.045
		r.muteIcon.Resize(fyne.NewSize(icon, icon))
		r.muteIcon.Move(fyne.NewPos(g.w-icon*1.4, g.h-icon*1.4))
		r.muteIcon.Show()
	} else {
		r.muteIcon.Hide()
	}
	r.muteIcon.Refresh()
}

// refreshBanner shows the centre-screen prompt for the current state.
func (r *boardRenderer) refreshBanner() {
	g := r.game
	r.overlay.Hide()
	var title, sub string
	switch g.state {
	case StateAttract:
		title = "STARBREAKER"
		sub = "Rotate: Left / Right     Thrust: Up     Fire: Space     Hyperspace: H\nPress Space or click to play"
	case StatePaused:
		r.overlay.Show()
		title = "PAUSED"
		sub = "Press P to resume"
	case StateGameOver:
		r.overlay.Show()
		title = "GAME OVER"
		sub = fmt.Sprintf("Score %d   -   reached wave %d\nPress Space or click to play again", g.score, g.wave)
	default:
		r.title.Hide()
		for _, t := range r.subtitle {
			t.Hide()
		}
		return
	}
	r.title.Show()
	r.title.Text = title
	r.title.TextSize = g.h * 0.075
	r.title.Resize(fyne.NewSize(g.w, r.title.TextSize*1.4))
	r.title.Move(fyne.NewPos(0, g.h*0.34))

	// canvas.Text is single-line, so render each "\n"-separated line separately.
	subLines := strings.Split(sub, "\n")
	subSize := g.h * 0.026
	top := g.h*0.34 + r.title.TextSize*1.6
	for i, t := range r.subtitle {
		line := ""
		if i < len(subLines) {
			line = subLines[i]
		}
		t.Text = line
		t.TextSize = subSize
		t.Resize(fyne.NewSize(g.w, subSize*1.4))
		t.Move(fyne.NewPos(0, top+float32(i)*subSize*1.6))
		if line == "" {
			t.Hide()
		} else {
			t.Show()
		}
	}
}

// --- geometry & object helpers ---------------------------------------------

// wrapOffsets returns the set of position offsets at which an object centred at
// pos with the given radius should be drawn so it appears seamlessly across the
// toroidal edges (one entry when well inside, up to four near a corner).
func (g *Game) wrapOffsets(pos vec, radius float32) []vec {
	xs := []float32{0}
	ys := []float32{0}
	if pos.x < radius {
		xs = append(xs, g.w)
	} else if pos.x > g.w-radius {
		xs = append(xs, -g.w)
	}
	if pos.y < radius {
		ys = append(ys, g.h)
	} else if pos.y > g.h-radius {
		ys = append(ys, -g.h)
	}
	offs := make([]vec, 0, len(xs)*len(ys))
	for _, x := range xs {
		for _, y := range ys {
			offs = append(offs, vec{x, y})
		}
	}
	return offs
}

// transform maps a unit shape to world space: scaled by radius, rotated by
// angle, translated to centre.
func transform(shape []vec, centre vec, radius, angle float32) []vec {
	out := make([]vec, len(shape))
	for i, v := range shape {
		out[i] = centre.add(v.scale(radius).rot(angle))
	}
	return out
}

// shift translates every point by off.
func shift(pts []vec, off vec) []vec {
	if off == (vec{}) {
		return pts
	}
	out := make([]vec, len(pts))
	for i, p := range pts {
		out[i] = p.add(off)
	}
	return out
}

// strokePoly builds a stroked, unfilled ArbitraryPolygon from world points by
// fitting it to a tight bounding box (points are object-relative).
func strokePoly(world []vec, col color.Color, width float32) *canvas.ArbitraryPolygon {
	minX, minY := world[0].x, world[0].y
	maxX, maxY := minX, minY
	for _, p := range world[1:] {
		minX, minY = float32(math.Min(float64(minX), float64(p.x))), float32(math.Min(float64(minY), float64(p.y)))
		maxX, maxY = float32(math.Max(float64(maxX), float64(p.x))), float32(math.Max(float64(maxY), float64(p.y)))
	}
	// Pad the box by the stroke so the outline isn't clipped at the quad edge.
	pad := width + 1
	minX, minY = minX-pad, minY-pad
	pts := make([]fyne.Position, len(world))
	for i, p := range world {
		pts[i] = fyne.NewPos(p.x-minX, p.y-minY)
	}
	poly := canvas.NewArbitraryPolygon(pts, color.Transparent)
	poly.StrokeColor = col
	poly.StrokeWidth = width
	poly.Move(fyne.NewPos(minX, minY))
	poly.Resize(fyne.NewSize(maxX-minX+pad, maxY-minY+pad))
	return poly
}

// line builds a neon line segment between two world points.
func line(a, b vec, col color.Color, width float32) *canvas.Line {
	l := canvas.NewLine(col)
	l.StrokeWidth = width
	l.Position1 = fyne.NewPos(a.x, a.y)
	l.Position2 = fyne.NewPos(b.x, b.y)
	return l
}

func newText(col color.Color, align fyne.TextAlign, bold bool) *canvas.Text {
	t := canvas.NewText("", col)
	t.Alignment = align
	t.TextStyle.Bold = bold
	return t
}

func canvasContainer() *fyne.Container { return container.NewWithoutLayout() }

func clampF(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
