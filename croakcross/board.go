package croakcross

import (
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
	"github.com/xdung24/games/internal/navigation"
)

// Theme colours for the board.
var (
	colNight    = color.NRGBA{0x0B, 0x1B, 0x2B, 0xFF} // outside the playfield and behind the HUD
	colGrass    = color.NRGBA{0x4C, 0x1D, 0x95, 0xFF} // the arcade's purple safe strips
	colRoad     = color.NRGBA{0x1F, 0x29, 0x37, 0xFF}
	colLaneLine = color.NRGBA{0xE5, 0xE7, 0xEB, 0x55}
	colRiver    = color.NRGBA{0x14, 0x2F, 0x6E, 0xFF}
	colBank     = color.NRGBA{0x16, 0x65, 0x34, 0xFF} // hedge along the home row
	colBay      = color.NRGBA{0x0E, 0x24, 0x52, 0xFF} // water inside each home bay
	colLily     = color.NRGBA{0x22, 0xC5, 0x5E, 0x66} // lily pad hint in an empty bay

	colFrog     = color.NRGBA{0x4A, 0xDE, 0x80, 0xFF}
	colFrogDark = color.NRGBA{0x16, 0xA3, 0x4A, 0xFF}
	colEye      = color.NRGBA{0xFE, 0xF0, 0x8A, 0xFF}
	colPupil    = color.NRGBA{0x11, 0x18, 0x27, 0xFF}

	colLogDark     = color.NRGBA{0x6B, 0x2F, 0x0A, 0xFF}
	colTurtleLight = color.NRGBA{0x86, 0xEF, 0xAC, 0xFF}
	colWindow      = color.NRGBA{0x1E, 0x29, 0x3B, 0xDD}
	colStripe      = color.NRGBA{0xEF, 0x44, 0x44, 0xFF}
	colTruckCab    = color.NRGBA{0x1D, 0x4E, 0xD8, 0xFF}
	colHeadlight   = color.NRGBA{0xFE, 0xF9, 0xC3, 0xFF}

	colFly     = color.NRGBA{0x6D, 0x28, 0xD9, 0xFF}
	colWing    = color.NRGBA{0xFF, 0xFF, 0xFF, 0x99}
	colCroc    = color.NRGBA{0x3F, 0x6F, 0x1F, 0xFF}
	colCrocEye = color.NRGBA{0xFD, 0xE0, 0x47, 0xFF}
	colTeeth   = color.NRGBA{0xF8, 0xFA, 0xFC, 0xFF}

	colSplat  = color.NRGBA{0xEF, 0x44, 0x44, 0xFF}
	colRipple = color.NRGBA{0xBF, 0xDB, 0xFE, 0xAA}

	colTimer    = color.NRGBA{0x4A, 0xDE, 0x80, 0xFF}
	colTimerLow = color.NRGBA{0xEF, 0x44, 0x44, 0xFF}
	colTimerBg  = color.NRGBA{0xFF, 0xFF, 0xFF, 0x22}

	colText      = color.NRGBA{0xE2, 0xE8, 0xF0, 0xFF}
	colDim       = color.NRGBA{0x94, 0xA3, 0xB8, 0xFF}
	colOver      = color.NRGBA{0x0B, 0x1B, 0x2B, 0xC8} // translucent veil for pause/over screens
	colOverLight = color.NRGBA{0x0B, 0x1B, 0x2B, 0x60} // lighter veil so the title screen stays lively
)

// swipeThreshold is how far (in points) a drag must travel to count as a hop.
const swipeThreshold = 18

// Board is the interactive play surface: a focusable custom widget that
// renders the game and feeds player input into the model.
type Board struct {
	widget.BaseWidget
	game     *Game
	renderer *boardRenderer // cached so the loop can reposition without a full Refresh

	dragX, dragY float32 // accumulated finger travel for swipe detection

	onToggleMute func() // toggles sound muting; wired up by main, nil = no-op
	onBack       func()
	escape       navigation.EscapeHandler
}

func NewBoard(g *Game) *Board {
	b := &Board{game: g}
	b.ExtendBaseWidget(b)
	return b
}

// MoveDynamic repositions only the things that move during normal play
// (traffic, platforms, the frog and the clock) without a full Refresh.
func (b *Board) MoveDynamic() {
	if b.renderer != nil {
		b.renderer.moveDynamic()
	}
}

// --- input: pointer (click and touch) ---

func (b *Board) MouseIn(*desktop.MouseEvent)    {}
func (b *Board) MouseOut()                      {}
func (b *Board) MouseMoved(*desktop.MouseEvent) {}

// Tapped hops the frog towards the tap while playing (whichever of up, down,
// left or right the tap lies in relative to the frog); otherwise it runs the
// primary action, like Space.
func (b *Board) Tapped(e *fyne.PointEvent) {
	b.escape.Reset()
	if b.game.state != StatePlaying || b.renderer == nil {
		b.activate()
		return
	}
	fx, fy := b.renderer.frogCentre()
	dx, dy := e.Position.X-fx, e.Position.Y-fy
	switch {
	case abs(dx) > abs(dy) && dx < 0:
		b.game.Hop(DirLeft)
	case abs(dx) > abs(dy):
		b.game.Hop(DirRight)
	case dy > 0:
		b.game.Hop(DirDown)
	default:
		b.game.Hop(DirUp)
	}
}

// Dragged accumulates finger travel; DragEnd turns it into a swipe hop.
func (b *Board) Dragged(e *fyne.DragEvent) {
	b.escape.Reset()
	b.dragX += e.Dragged.DX
	b.dragY += e.Dragged.DY
}

func (b *Board) DragEnd() {
	dx, dy := b.dragX, b.dragY
	b.dragX, b.dragY = 0, 0
	if abs(dx) < swipeThreshold && abs(dy) < swipeThreshold {
		return
	}
	switch {
	case abs(dx) > abs(dy) && dx < 0:
		b.game.Hop(DirLeft)
	case abs(dx) > abs(dy):
		b.game.Hop(DirRight)
	case dy > 0:
		b.game.Hop(DirDown)
	default:
		b.game.Hop(DirUp)
	}
}

// activate runs the primary action: start from the title, toggle pause during
// play, or begin a fresh game once it's over.
func (b *Board) activate() {
	switch b.game.state {
	case StateReady:
		b.game.Start()
	case StatePlaying, StatePaused:
		b.game.TogglePause()
	case StateGameOver, StateWon:
		b.game.Restart()
	}
}

// --- input: keyboard ---

func (b *Board) FocusGained()   {}
func (b *Board) FocusLost()     { b.escape.Reset() }
func (b *Board) TypedRune(rune) {}

func (b *Board) TypedKey(e *fyne.KeyEvent) {
	if b.escape.TypedKey(e.Name, func() {
		b.game.Pause()
		b.Refresh()
	}, b.onBack) {
		return
	}
	switch e.Name {
	case fyne.KeyUp, fyne.KeyW:
		b.game.Hop(DirUp)
	case fyne.KeyDown, fyne.KeyS:
		b.game.Hop(DirDown)
	case fyne.KeyLeft, fyne.KeyA:
		b.game.Hop(DirLeft)
	case fyne.KeyRight, fyne.KeyD:
		b.game.Hop(DirRight)
	case fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter:
		b.activate()
	case fyne.KeyP:
		b.game.TogglePause()
	case fyne.KeyR:
		b.game.Restart()
	case fyne.KeyM:
		if b.onToggleMute != nil {
			b.onToggleMute()
		}
	}
}

func (b *Board) KeyUp(e *fyne.KeyEvent) {
	b.escape.KeyUp(e.Name)
}

// CreateRenderer wires up the persistent canvas objects.
func (b *Board) CreateRenderer() fyne.WidgetRenderer {
	r := &boardRenderer{board: b, game: b.game}

	r.bg = canvas.NewRectangle(colNight)
	r.grassLow = canvas.NewRectangle(colGrass)
	r.road = canvas.NewRectangle(colRoad)
	r.median = canvas.NewRectangle(colGrass)
	r.river = canvas.NewRectangle(colRiver)
	r.bank = canvas.NewRectangle(colBank)
	for i := range r.bays {
		r.bays[i] = canvas.NewRectangle(colBay)
		r.lilies[i] = canvas.NewCircle(colLily)
	}
	// Dashed centre lines between the road lanes.
	for i := 0; i < (medianRow-startRow-2)*cols; i++ {
		r.dashes = append(r.dashes, canvas.NewRectangle(colLaneLine))
	}

	r.frog = newFrogSprite()
	for i := range r.homeFrogs {
		r.homeFrogs[i] = newFrogSprite()
	}
	r.fly = newFlySprite()
	r.croc = newCrocSprite()

	r.splat = canvas.NewRectangle(colSplat)
	for i := range r.ripples {
		c := canvas.NewCircle(color.Transparent)
		c.StrokeColor = colRipple
		c.StrokeWidth = 2
		r.ripples[i] = c
	}

	r.score = canvas.NewText("", colText)
	r.score.TextStyle.Bold = true
	r.level = canvas.NewText("", colDim)
	r.level.TextStyle.Bold = true
	r.level.Alignment = fyne.TextAlignCenter
	r.lives = canvas.NewText("", colFrog)
	r.lives.TextStyle.Bold = true
	r.lives.Alignment = fyne.TextAlignTrailing

	r.timeLabel = canvas.NewText("TIME", colText)
	r.timeLabel.TextStyle.Bold = true
	r.timerBack = canvas.NewRectangle(colTimerBg)
	r.timerBar = canvas.NewRectangle(colTimer)

	r.over = canvas.NewRectangle(colOver)
	r.over.Hide()
	r.title = canvas.NewText("", colText)
	r.title.TextStyle.Bold = true
	r.title.Alignment = fyne.TextAlignCenter
	r.subtitle = canvas.NewText("", colDim)
	r.subtitle.Alignment = fyne.TextAlignCenter

	b.renderer = r
	return r
}

// --- sprites ---

// part is one canvas object positioned relative to its sprite's origin. All
// measurements are in cell units so a sprite scales with the board.
type part struct {
	obj          fyne.CanvasObject
	dx, dy, w, h float32
	radius       float32 // corner radius for rectangles, in cells
}

// sprite is a small group of canvas objects moved as one. Parts are sized only
// when the board is laid out (resizing regenerates textures) and merely moved
// each frame, which is cheap.
type sprite struct {
	parts []part
	base  []part // untransformed parts, kept so the frog can be re-faced
}

func (s *sprite) add(obj fyne.CanvasObject, dx, dy, w, h, radius float32) {
	s.parts = append(s.parts, part{obj, dx, dy, w, h, radius})
}

func (s *sprite) rect(col color.Color, dx, dy, w, h, radius float32) {
	s.add(canvas.NewRectangle(col), dx, dy, w, h, radius)
}

func (s *sprite) circle(col color.Color, dx, dy, d float32) {
	s.add(canvas.NewCircle(col), dx, dy, d, d, 0)
}

// layout sizes every part for the current cell size.
func (s *sprite) layout(cell float32) {
	for _, p := range s.parts {
		p.obj.Resize(fyne.NewSize(p.w*cell, p.h*cell))
		if rect, ok := p.obj.(*canvas.Rectangle); ok {
			rect.CornerRadius = p.radius * cell
		}
	}
}

// move places the sprite's origin at the pixel position (x, y).
func (s *sprite) move(x, y, cell float32) {
	for _, p := range s.parts {
		p.obj.Move(fyne.NewPos(x+p.dx*cell, y+p.dy*cell))
	}
}

func (s *sprite) show() {
	for _, p := range s.parts {
		p.obj.Show()
	}
}

func (s *sprite) hide() {
	for _, p := range s.parts {
		p.obj.Hide()
	}
}

// refresh asks the canvas to regenerate every part (after a colour change).
func (s *sprite) refresh() {
	for _, p := range s.parts {
		canvas.Refresh(p.obj)
	}
}

// mirror flips the sprite horizontally within a box width cells wide, so a
// vehicle drawn facing right can face left.
func (s *sprite) mirror(width float32) {
	for i := range s.parts {
		s.parts[i].dx = width - s.parts[i].dx - s.parts[i].w
	}
}

// newFrogSprite builds a frog facing up inside a 1x1 cell.
func newFrogSprite() *sprite {
	s := &sprite{}
	// Legs first so the body overlaps them.
	s.rect(colFrogDark, 0.02, 0.18, 0.24, 0.2, 0.08)
	s.rect(colFrogDark, 0.74, 0.18, 0.24, 0.2, 0.08)
	s.rect(colFrogDark, 0.02, 0.62, 0.24, 0.22, 0.08)
	s.rect(colFrogDark, 0.74, 0.62, 0.24, 0.22, 0.08)
	s.rect(colFrog, 0.17, 0.2, 0.66, 0.62, 0.2)
	s.circle(colEye, 0.22, 0.08, 0.22)
	s.circle(colEye, 0.56, 0.08, 0.22)
	s.circle(colPupil, 0.28, 0.12, 0.1)
	s.circle(colPupil, 0.62, 0.12, 0.1)
	s.base = append([]part(nil), s.parts...)
	return s
}

// face rotates the frog's parts so its eyes point in direction d. The frog is
// drawn facing up; the other directions are reflections of that layout.
func (s *sprite) face(d Dir, cell float32) {
	for i, b := range s.base {
		p := b
		switch d {
		case DirDown:
			p.dy = 1 - b.dy - b.h
		case DirLeft:
			p.dx, p.dy, p.w, p.h = b.dy, b.dx, b.h, b.w
		case DirRight:
			p.dx, p.dy, p.w, p.h = 1-b.dy-b.h, b.dx, b.h, b.w
		}
		s.parts[i] = p
	}
	s.layout(cell)
}

func newFlySprite() *sprite {
	s := &sprite{}
	s.circle(colWing, 0.18, 0.3, 0.28)
	s.circle(colWing, 0.54, 0.3, 0.28)
	s.circle(colFly, 0.36, 0.36, 0.28)
	return s
}

func newCrocSprite() *sprite {
	s := &sprite{}
	s.rect(colCroc, 0.02, 0.3, 0.96, 0.42, 0.12)
	s.rect(colTeeth, 0.06, 0.56, 0.4, 0.07, 0)
	s.circle(colCrocEye, 0.7, 0.22, 0.16)
	return s
}

// newMoverSprite draws a vehicle, log or turtle group for m. Vehicles are drawn
// facing right and mirrored for leftward lanes.
func newMoverSprite(m *Mover) *sprite {
	s := &sprite{}
	switch m.kind {
	case KindCar:
		s.rect(m.col, 0.06, 0.2, 0.88, 0.6, 0.12)
		s.rect(colWindow, 0.4, 0.28, 0.3, 0.44, 0.05)
		s.rect(colHeadlight, 0.88, 0.28, 0.08, 0.12, 0.02)
		s.rect(colHeadlight, 0.88, 0.6, 0.08, 0.12, 0.02)
	case KindRacer:
		s.rect(m.col, 0.04, 0.3, 0.92, 0.4, 0.14)
		s.rect(colStripe, 0.12, 0.46, 0.76, 0.08, 0)
		s.rect(colWindow, 0.5, 0.36, 0.22, 0.28, 0.05)
	case KindTractor:
		s.rect(m.col, 0.1, 0.3, 0.8, 0.5, 0.08)
		s.rect(colWindow, 0.18, 0.14, 0.34, 0.3, 0.05)
		s.rect(colPupil, 0.08, 0.72, 0.84, 0.14, 0.06)
	case KindTruck:
		s.rect(m.col, 0.05, 0.16, 1.3, 0.68, 0.08)
		s.rect(colTruckCab, 1.4, 0.22, 0.56, 0.56, 0.1)
		s.rect(colWindow, 1.66, 0.3, 0.22, 0.4, 0.04)
	case KindLog:
		s.rect(m.col, 0, 0.16, m.w, 0.68, 0.3)
		s.rect(colLogDark, 0.3, 0.32, m.w-0.6, 0.08, 0.04)
		s.rect(colLogDark, 0.5, 0.6, m.w-1.0, 0.08, 0.04)
	case KindTurtle:
		for i := float32(0); i < m.w; i++ {
			s.circle(m.col, i+0.1, 0.1, 0.8)
			s.circle(colTurtleLight, i+0.3, 0.3, 0.4)
		}
	}
	if m.kind.IsVehicle() && m.speed < 0 {
		s.mirror(m.w)
	}
	return s
}

// setTurtleAlpha fades a turtle group as it dives. Alpha is quantised so the
// textures are regenerated only a handful of times per transition.
func setTurtleAlpha(s *sprite, surface float32) {
	a := uint8(math.Round(float64(surface)*4) * 63.75)
	changed := false
	for i, p := range s.parts {
		c := p.obj.(*canvas.Circle)
		base := colTurtle
		if i%2 == 1 {
			base = colTurtleLight
		}
		want := color.NRGBA{base.R, base.G, base.B, a}
		if c.FillColor != want {
			c.FillColor = want
			changed = true
		}
	}
	if changed {
		s.refresh()
	}
}

// --- renderer ---

// boardRenderer maps game state onto canvas objects.
type boardRenderer struct {
	board *Board
	game  *Game

	cell, ox, oy float32 // cell size and the playfield's top-left corner, in points

	bg                             *canvas.Rectangle
	grassLow, road, median, river  *canvas.Rectangle
	bank                           *canvas.Rectangle
	bays                           [numHomes]*canvas.Rectangle
	lilies                         [numHomes]*canvas.Circle
	dashes                         []*canvas.Rectangle
	movers                         []*sprite
	moverGen                       int // game.gen the sprite pool was built for
	frog                           *sprite
	frogFacing                     Dir
	homeFrogs                      [numHomes]*sprite
	fly, croc                      *sprite
	splat                          *canvas.Rectangle
	ripples                        [3]*canvas.Circle
	score, level, lives, timeLabel *canvas.Text
	timerBack, timerBar            *canvas.Rectangle
	over                           *canvas.Rectangle
	title, subtitle                *canvas.Text
	objs                           []fyne.CanvasObject // cached object list, rebuilt with the sprite pool
}

// px converts a column (cells) to a horizontal point position.
func (r *boardRenderer) px(x float32) float32 { return r.ox + x*r.cell }

// rowTop returns the top edge of a (possibly fractional) row. Row 0 is at the
// bottom of the playfield and the HUD occupies the band above the home row.
func (r *boardRenderer) rowTop(row float32) float32 {
	return r.oy + r.cell*(1+float32(rows-1)-row)
}

// frogCentre returns the frog's on-screen centre, used to aim taps.
func (r *boardRenderer) frogCentre() (x, y float32) {
	fx, frow, _ := r.game.FrogVisual()
	return r.px(fx), r.rowTop(frow) + r.cell/2
}

func (r *boardRenderer) MinSize() fyne.Size { return fyne.NewSize(cols*24, (rows+2)*24) }

func (r *boardRenderer) Destroy() {}

// Layout fits the playfield (plus a HUD row above and a clock row below) into
// the widget, centred, and re-sizes every object for the new cell size.
func (r *boardRenderer) Layout(size fyne.Size) {
	r.cell = min(size.Width/cols, size.Height/(rows+2))
	r.ox = (size.Width - r.cell*cols) / 2
	r.oy = (size.Height - r.cell*(rows+2)) / 2
	c := r.cell
	w := c * cols

	r.bg.Resize(size)
	r.over.Resize(size)

	r.grassLow.Move(fyne.NewPos(r.ox, r.rowTop(startRow)))
	r.grassLow.Resize(fyne.NewSize(w, c))
	r.road.Move(fyne.NewPos(r.ox, r.rowTop(medianRow-1)))
	r.road.Resize(fyne.NewSize(w, c*(medianRow-startRow-1)))
	r.median.Move(fyne.NewPos(r.ox, r.rowTop(medianRow)))
	r.median.Resize(fyne.NewSize(w, c))
	r.river.Move(fyne.NewPos(r.ox, r.rowTop(homeRow-1)))
	r.river.Resize(fyne.NewSize(w, c*(homeRow-medianRow-1)))
	r.bank.Move(fyne.NewPos(r.ox, r.rowTop(homeRow)))
	r.bank.Resize(fyne.NewSize(w, c))
	for i, cx := range homeCenters {
		r.bays[i].Move(fyne.NewPos(r.px(cx-0.5), r.rowTop(homeRow)))
		r.bays[i].Resize(fyne.NewSize(c, c*0.92))
		r.lilies[i].Move(fyne.NewPos(r.px(cx-0.3), r.rowTop(homeRow)+c*0.18))
		r.lilies[i].Resize(fyne.NewSize(c*0.6, c*0.6))
	}
	for i, d := range r.dashes {
		boundary := i/cols + 1 // between road rows boundary and boundary+1
		col := i % cols
		d.Move(fyne.NewPos(r.px(float32(col)+0.25), r.rowTop(float32(boundary))-c*0.03))
		d.Resize(fyne.NewSize(c*0.5, c*0.06))
	}

	r.syncMovers()
	for _, s := range r.movers {
		s.layout(c)
	}
	r.frog.face(r.frogFacing, c)
	for _, s := range r.homeFrogs {
		s.face(DirUp, c)
	}
	r.fly.layout(c)
	r.croc.layout(c)

	r.splat.Resize(fyne.NewSize(c*0.9, c*0.3))
	r.splat.CornerRadius = c * 0.15
	for i, rp := range r.ripples {
		d := c * (0.4 + 0.4*float32(i))
		rp.Resize(fyne.NewSize(d, d))
	}

	hud := c * 0.5
	r.score.TextSize = hud
	r.score.Move(fyne.NewPos(r.px(0.3), r.oy+c*0.22))
	r.level.TextSize = hud
	r.level.Resize(fyne.NewSize(w, hud*1.4))
	r.level.Move(fyne.NewPos(r.ox, r.oy+c*0.22))
	r.lives.TextSize = hud
	r.lives.Resize(fyne.NewSize(w-r.px(0.3)+r.ox, hud*1.4))
	r.lives.Move(fyne.NewPos(r.ox, r.oy+c*0.22))

	bottom := r.rowTop(startRow) + c
	r.timeLabel.TextSize = hud
	r.timeLabel.Move(fyne.NewPos(r.px(cols-2.1), bottom+c*0.22))
	r.timerBack.Move(fyne.NewPos(r.px(0.5), bottom+c*0.32))
	r.timerBack.Resize(fyne.NewSize(c*(cols-3), c*0.36))

	r.title.TextSize = c * 1.1
	r.title.Resize(fyne.NewSize(size.Width, r.title.TextSize*1.4))
	r.title.Move(fyne.NewPos(0, r.rowTop(medianRow)-c*1.2))
	r.subtitle.TextSize = c * 0.42
	r.subtitle.Resize(fyne.NewSize(size.Width, r.subtitle.TextSize*1.4))
	r.subtitle.Move(fyne.NewPos(0, r.rowTop(medianRow)+c*0.5))

	r.Refresh()
}

// syncMovers rebuilds the sprite pool whenever the game builds a new lane set.
func (r *boardRenderer) syncMovers() {
	if r.objs != nil && r.moverGen == r.game.gen {
		return
	}
	r.moverGen = r.game.gen
	r.movers = r.movers[:0]
	for _, m := range r.game.movers {
		s := newMoverSprite(m)
		s.layout(r.cell)
		r.movers = append(r.movers, s)
	}
	r.rebuildObjects()
}

// rebuildObjects caches the draw list, back to front.
func (r *boardRenderer) rebuildObjects() {
	objs := []fyne.CanvasObject{r.bg, r.grassLow, r.road, r.median, r.river, r.bank}
	for i := range r.bays {
		objs = append(objs, r.bays[i], r.lilies[i])
	}
	for _, d := range r.dashes {
		objs = append(objs, d)
	}
	for _, s := range r.movers {
		for _, p := range s.parts {
			objs = append(objs, p.obj)
		}
	}
	for _, s := range r.homeFrogs {
		for _, p := range s.parts {
			objs = append(objs, p.obj)
		}
	}
	for _, s := range []*sprite{r.fly, r.croc, r.frog} {
		for _, p := range s.parts {
			objs = append(objs, p.obj)
		}
	}
	objs = append(objs, r.splat)
	for _, rp := range r.ripples {
		objs = append(objs, rp)
	}
	objs = append(objs, r.score, r.level, r.lives, r.timeLabel, r.timerBack, r.timerBar,
		r.over, r.title, r.subtitle)
	r.objs = objs
}

func (r *boardRenderer) Objects() []fyne.CanvasObject {
	r.syncMovers()
	return r.objs
}

// Refresh restyles everything from the model: homes, HUD text, the bay visitor
// and the banner, then does the per-frame repositioning too.
func (r *boardRenderer) Refresh() {
	g := r.game
	r.syncMovers()
	c := r.cell

	for i, filled := range g.homes {
		if filled {
			r.homeFrogs[i].move(r.px(homeCenters[i]-0.5), r.rowTop(homeRow), c)
			r.homeFrogs[i].show()
			r.lilies[i].Hide()
		} else {
			r.homeFrogs[i].hide()
			r.lilies[i].Show()
		}
	}

	r.fly.hide()
	r.croc.hide()
	if g.visitor.kind != VisitorNone {
		s := r.fly
		if g.visitor.kind == VisitorCroc {
			s = r.croc
		}
		s.move(r.px(homeCenters[g.visitor.bay]-0.5), r.rowTop(homeRow), c)
		s.show()
	}

	setText(r.score, fmt.Sprintf("SCORE  %d", g.score))
	setText(r.level, fmt.Sprintf("LEVEL %d", g.level))
	setText(r.lives, lifeIcons(g.lives))

	r.refreshBanner()
	r.moveDynamic()
}

// moveDynamic repositions the traffic, platforms, frog and clock. Textures are
// unchanged (only positions moved), so this is far cheaper than Refresh.
func (r *boardRenderer) moveDynamic() {
	g := r.game
	c := r.cell

	for i, m := range g.movers {
		if i >= len(r.movers) {
			break
		}
		s := r.movers[i]
		s.move(r.px(m.x), r.rowTop(float32(m.row)), c)
		if m.dives {
			setTurtleAlpha(s, m.surface())
		}
	}

	r.moveFrog()

	// Clock: a bar that shrinks towards the TIME label and reddens when short.
	frac := clamp(g.timeLeft/frogTime, 0, 1)
	full := c * (cols - 3)
	bw := full * frac
	r.timerBar.Move(fyne.NewPos(r.px(0.5)+full-bw, r.timerBack.Position().Y))
	r.timerBar.Resize(fyne.NewSize(bw, c*0.36))
	want := colTimer
	if g.timeLeft <= 10 {
		want = colTimerLow
	}
	if r.timerBar.FillColor != want {
		r.timerBar.FillColor = want
		canvas.Refresh(r.timerBar)
	}

	canvas.Refresh(r.bg)
}

// moveFrog places the live frog or, while dying, its splat or ripples.
func (r *boardRenderer) moveFrog() {
	g := r.game
	c := r.cell
	fx, frow, arc := g.FrogVisual()
	x := r.px(fx - 0.5)
	y := r.rowTop(frow) - arc*c*0.15

	r.splat.Hide()
	for _, rp := range r.ripples {
		rp.Hide()
	}

	showFrog := g.state != StateLevelClear && g.state != StateWon
	if g.state == StateDying && g.stateT >= hopDuration {
		showFrog = false
		cx, cy := r.px(g.frog.x), r.rowTop(float32(g.frog.row))+c/2
		if g.deathCause == DeathDrown {
			// Ripples spread outwards one after another.
			for i, rp := range r.ripples {
				if g.stateT > hopDuration+0.2*float32(i) {
					d := rp.Size().Width
					rp.Move(fyne.NewPos(cx-d/2, cy-d/2))
					rp.Show()
				}
			}
		} else {
			r.splat.Move(fyne.NewPos(cx-c*0.45, cy-c*0.15))
			r.splat.Show()
		}
	}

	if !showFrog {
		r.frog.hide()
		return
	}
	if r.frogFacing != g.frog.facing {
		r.frogFacing = g.frog.facing
		r.frog.face(r.frogFacing, c)
	}
	r.frog.move(x, y, c)
	r.frog.show()
}

// refreshBanner shows the centre-screen prompt appropriate to the game state.
func (r *boardRenderer) refreshBanner() {
	g := r.game
	var title, sub string
	r.over.Hide()
	veil := colOver
	switch g.state {
	case StateReady:
		veil = colOverLight
		r.over.Show()
		title = "CroakCross"
		sub = "Arrow keys, tap or swipe to hop"
	case StatePaused:
		r.over.Show()
		title = "Paused"
		sub = "Press Space to resume"
	case StateLevelClear:
		title = fmt.Sprintf("Level %d cleared!", g.level)
		sub = "Bonus 1000"
	case StateGameOver:
		r.over.Show()
		title = "Game Over"
		sub = fmt.Sprintf("Score %d  ·  Space or tap to play again", g.score)
	case StateWon:
		r.over.Show()
		title = "You Win!"
		sub = fmt.Sprintf("All %d levels crossed  ·  Score %d  ·  Space to play again", maxLevel, g.score)
	default:
		r.title.Hide()
		r.subtitle.Hide()
		return
	}
	if r.over.FillColor != veil {
		r.over.FillColor = veil
		canvas.Refresh(r.over)
	}
	setText(r.title, title)
	setText(r.subtitle, sub)
	r.title.Show()
	r.subtitle.Show()
}

// setText updates a canvas text, regenerating it only when the string changed.
func setText(t *canvas.Text, s string) {
	if t.Text == s {
		return
	}
	t.Text = s
	canvas.Refresh(t)
}

func lifeIcons(n int) string {
	s := ""
	for i := 0; i < n; i++ {
		s += "● "
	}
	if s == "" {
		s = "—"
	}
	return s
}
