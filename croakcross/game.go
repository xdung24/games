package croakcross

import (
	"image/color"
	"math"
	"math/rand"
)

// GameState models the high-level phase of play.
type GameState int

const (
	StateReady      GameState = iota // title screen; traffic flows, frog waits for the first hop
	StatePlaying                     // frog in play
	StatePaused                      // frozen by the player
	StateDying                       // death animation playing
	StateLevelClear                  // all five homes filled; brief celebration before the next level
	StateGameOver                    // out of lives
	StateWon                         // every level cleared
)

// Board geometry. The model works entirely in cell units: the playfield is
// cols cells wide and rows cells tall, with row 0 (the start bank) at the
// bottom and the home row at the top. The renderer maps cells to pixels.
const (
	cols      = 13
	rows      = 13
	startRow  = 0  // safe grass at the bottom
	medianRow = 6  // safe strip between road and river
	homeRow   = 12 // the bank with the five home bays
	numHomes  = 5

	startLives     = 3
	frogTime       = 30.0 // seconds to get one frog home
	hopDuration    = 0.1  // seconds the hop animation takes
	dyingDuration  = 1.0  // seconds the splat / splash is shown
	clearDuration  = 1.6  // seconds the level-cleared banner is shown
	extraLifeEvery = 10000
	frogHalfW      = 0.4  // half-width of the frog's collision box, cells
	bayTolerance   = 0.6  // how far off a bay's centre a landing may be, cells
	divePeriod     = 7.0  // seconds for one surface/dive cycle of a diving turtle
	rideSlack      = 0.15 // how far past a platform's edge the frog can still cling, cells
)

// maxLevel is the number of rounds; clearing the last one wins the game.
const maxLevel = 10

// homeCenters are the column centres of the five home bays. They sit half a
// cell off the column grid, exactly like the arcade original, which is why a
// frog carried along by a log is often lined up better than one hopping
// straight off the median.
var homeCenters = [numHomes]float32{1, 3.5, 6, 8.5, 11}

// Dir is a hop direction.
type Dir int

const (
	DirUp Dir = iota
	DirDown
	DirLeft
	DirRight
)

// Sound names an audible game event. The model only emits these through a
// callback, so it stays free of any audio dependency (and the tests stay silent).
type Sound int

const (
	SoundHop Sound = iota
	SoundHome
	SoundFly
	SoundSquash
	SoundSplash
	SoundTimeWarning
	SoundExtraLife
	SoundLevelUp
	SoundGameOver
	SoundWin
)

// DeathCause records how the frog met its end, which picks the animation.
type DeathCause int

const (
	DeathSquash DeathCause = iota // run over on the road, or splatted against the home bank
	DeathDrown                    // fell in the river or was carried off the edge
	DeathTime                     // the clock ran out
)

// Kind is the type of a moving object in a lane.
type Kind int

const (
	KindCar     Kind = iota // small car
	KindTruck               // two cells long
	KindRacer               // fast sports car
	KindTractor             // slow and chunky
	KindLog                 // a floating platform
	KindTurtle              // a group of turtles, some of which dive
)

// IsVehicle reports whether the kind kills the frog on contact.
func (k Kind) IsVehicle() bool { return k <= KindTractor }

// Mover is one car, truck, log or turtle group sliding along a lane. Each lane
// holds several movers spaced evenly around a wrap period, so a mover that
// leaves one side reappears on the other after a consistent gap.
type Mover struct {
	kind   Kind
	row    int
	x, w   float32 // left edge and width, cells
	speed  float32 // cells per second; negative moves leftward
	period float32 // distance travelled before the lane pattern repeats
	col    color.Color

	// Turtles only.
	dives bool    // this group periodically submerges
	diveT float32 // position in the dive cycle, seconds
}

// advance slides the mover and wraps it around once it has fully left the board.
func (m *Mover) advance(dt float32) {
	m.x += m.speed * dt
	if m.speed > 0 && m.x > cols+1 {
		m.x -= m.period
	} else if m.speed < 0 && m.x+m.w < -1 {
		m.x += m.period
	}
	if m.dives {
		m.diveT += dt
		for m.diveT >= divePeriod {
			m.diveT -= divePeriod
		}
	}
}

// surface is 1 while a turtle group is afloat, 0 when fully submerged, and in
// between while sinking or rising. Non-diving movers are always afloat.
func (m *Mover) surface() float32 {
	if !m.dives {
		return 1
	}
	t := m.diveT
	switch {
	case t < 4.2:
		return 1
	case t < 5.0:
		return 1 - (t-4.2)/0.8
	case t < 6.2:
		return 0
	default:
		return (t - 6.2) / 0.8
	}
}

// supports reports whether a frog can stand on the mover right now.
func (m *Mover) supports() bool {
	return m.kind == KindLog || (m.kind == KindTurtle && m.surface() > 0.35)
}

// laneDef describes one lane's traffic at level 1.
type laneDef struct {
	row     int
	kind    Kind
	speed   float32 // cells/s, negative = leftward
	w       float32 // width of each mover, cells
	n       int     // movers in the lane
	spacing float32 // distance between successive movers' left edges, cells
	col     color.Color
	dive    int // turtles: every dive-th group dives (0 = none)
}

// Lane colours.
var (
	colCarYellow = color.NRGBA{0xFA, 0xCC, 0x15, 0xFF}
	colCarPink   = color.NRGBA{0xF4, 0x72, 0xB6, 0xFF}
	colRacer     = color.NRGBA{0xF8, 0xFA, 0xFC, 0xFF}
	colTractor   = color.NRGBA{0xEA, 0x58, 0x0C, 0xFF}
	colTruck     = color.NRGBA{0x93, 0xC5, 0xFD, 0xFF}
	colLog       = color.NRGBA{0x92, 0x40, 0x0E, 0xFF}
	colTurtle    = color.NRGBA{0x15, 0x80, 0x3D, 0xFF}
)

// laneDefs lays out the classic ten lanes, bottom to top: five lanes of road
// traffic, then (above the median) five lanes of river platforms. Direction
// alternates so the frog has to read both ways.
var laneDefs = []laneDef{
	{row: 1, kind: KindCar, speed: -1.5, w: 1, n: 3, spacing: 5, col: colCarYellow},
	{row: 2, kind: KindTractor, speed: 1.1, w: 1, n: 3, spacing: 5.5, col: colTractor},
	{row: 3, kind: KindCar, speed: -2.0, w: 1, n: 3, spacing: 5, col: colCarPink},
	{row: 4, kind: KindRacer, speed: 3.4, w: 1, n: 1, spacing: 8, col: colRacer},
	{row: 5, kind: KindTruck, speed: -1.3, w: 2, n: 2, spacing: 8, col: colTruck},
	{row: 7, kind: KindTurtle, speed: -1.4, w: 3, n: 4, spacing: 4.5, col: colTurtle, dive: 2},
	{row: 8, kind: KindLog, speed: 1.2, w: 3, n: 3, spacing: 6, col: colLog},
	{row: 9, kind: KindLog, speed: 2.3, w: 6, n: 2, spacing: 9, col: colLog},
	{row: 10, kind: KindTurtle, speed: -1.7, w: 2, n: 4, spacing: 4, col: colTurtle, dive: 2},
	{row: 11, kind: KindLog, speed: 1.6, w: 4, n: 3, spacing: 6.5, col: colLog},
}

// Frog is the player. Hops are logically instant - the frog is at its new cell
// the moment the key is pressed, so collisions are always judged against where
// the player meant to be - while a short visual offset decays to make the move
// read as a hop rather than a teleport.
type Frog struct {
	x      float32 // centre column, cells (continuous: logs carry it)
	row    int
	facing Dir

	hopT    float32 // remaining hop animation time; > 0 while mid-hop
	hopDX   float32 // the hop's horizontal travel, cells
	hopDRow float32 // the hop's vertical travel, rows
}

// VisitorKind is what, if anything, is sitting in an empty home bay.
type VisitorKind int

const (
	VisitorNone VisitorKind = iota
	VisitorFly              // worth a bonus when caught
	VisitorCroc             // eats any frog that lands on it
)

// bayVisitor tracks the fly or crocodile that occasionally occupies a bay.
type bayVisitor struct {
	kind VisitorKind
	bay  int
	t    float32 // time left before the visitor leaves
	next float32 // time until the next visitor arrives (while kind is None)
}

// Render is how much work a frame needs: none, a cheap reposition, or a full redraw.
type Render int

const (
	RenderNone Render = iota // nothing changed; skip the frame entirely
	RenderMove               // only movers/frog/timer shifted; just reposition them
	RenderFull               // structural change; re-run the whole renderer
)

// Game holds all mutable play state. Every field is touched only from the UI
// goroutine (the ticker schedules updates via fyne.Do), so no locking is needed.
type Game struct {
	state GameState
	level int
	score int
	lives int

	nextExtra int // score at which the next extra life is awarded

	movers  []*Mover
	homes   [numHomes]bool
	visitor bayVisitor
	frog    Frog

	timeLeft   float32 // seconds left for this frog
	warned     bool    // low-time warning already sounded for this frog
	highestRow int     // furthest row reached this life (forward hops score once)

	stateT     float32    // time spent in the current timed state (dying / level clear)
	deathCause DeathCause // how the frog died, for the animation

	pending    Dir // a hop queued while the previous one was still animating
	hasPending bool

	// gen increments whenever the set of movers is rebuilt (each new level), so
	// the renderer knows when to rebuild its sprite pool.
	gen int

	render  Render
	onSound func(Sound) // optional sink for sound events; nil = silent
}

// NewGame builds a game at level 1 sitting on the title screen.
func NewGame() *Game {
	g := &Game{}
	g.startGame()
	g.state = StateReady
	return g
}

// needRender escalates the pending work for this frame to at least level.
func (g *Game) needRender(level Render) {
	if level > g.render {
		g.render = level
	}
}

// play emits a sound event if a sink is wired up.
func (g *Game) play(s Sound) {
	if g.onSound != nil {
		g.onSound(s)
	}
}

// startGame resets score and lives and builds the first level.
func (g *Game) startGame() {
	g.score = 0
	g.lives = startLives
	g.level = 1
	g.nextExtra = extraLifeEvery
	g.state = StatePlaying
	g.setupLevel()
}

// setupLevel populates every lane for g.level, empties the homes and puts a
// fresh frog on the start bank. Each level runs the traffic a little faster and
// turns more turtle groups into divers.
func (g *Game) setupLevel() {
	mul := 1 + 0.1*float32(g.level-1)

	g.movers = g.movers[:0]
	for _, d := range laneDefs {
		n := d.n
		if d.kind == KindRacer && g.level >= 4 {
			n = 2 // a second racer from level 4
		}
		dive := d.dive
		if dive > 0 && (d.row == 7 && g.level >= 3 || d.row == 10 && g.level >= 6) {
			dive = 1 // every group in the lane dives
		}

		period := float32(n) * d.spacing
		if minPeriod := float32(cols) + d.w + 2; period < minPeriod {
			period = minPeriod // always respawn fully off-screen
		}
		spacing := period / float32(n)
		start := rand.Float32() * period
		for i := 0; i < n; i++ {
			m := &Mover{kind: d.kind, row: d.row, w: d.w, speed: d.speed * mul, period: period, col: d.col}
			m.x = wrapX(start+float32(i)*spacing, m.w, period)
			if dive > 0 && i%dive == 0 {
				m.dives = true
				m.diveT = float32(math.Mod(float64(i)*1.9, divePeriod))
			}
			g.movers = append(g.movers, m)
		}
	}

	for i := range g.homes {
		g.homes[i] = false
	}
	g.visitor = bayVisitor{next: 5}
	g.gen++
	g.spawnFrog()
}

// wrapX normalises a lane position into the window [-w-1, period-w-1).
func wrapX(x, w, period float32) float32 {
	return float32(math.Mod(float64(x+w+1), float64(period))) - w - 1
}

// spawnFrog places a new frog on the start bank with a full clock.
func (g *Game) spawnFrog() {
	g.frog = Frog{x: float32(cols) / 2, row: startRow, facing: DirUp}
	g.timeLeft = frogTime
	g.warned = false
	g.highestRow = startRow
	g.hasPending = false
	g.needRender(RenderFull)
}

// Start leaves the title screen and begins play.
func (g *Game) Start() {
	if g.state == StateReady {
		g.state = StatePlaying
		g.needRender(RenderFull)
	}
}

// TogglePause flips between Playing and Paused (no-op in other states).
func (g *Game) TogglePause() {
	switch g.state {
	case StatePlaying:
		g.state = StatePaused
		g.needRender(RenderFull)
	case StatePaused:
		g.state = StatePlaying
		g.needRender(RenderFull)
	}
}

// Pause freezes play if a frog is in play. Unlike TogglePause it only ever
// pauses, never resumes.
func (g *Game) Pause() {
	if g.state == StatePlaying {
		g.state = StatePaused
		g.needRender(RenderFull)
	}
}

// Restart begins a brand new game.
func (g *Game) Restart() {
	g.startGame()
}

// Hop asks the frog to jump one cell. On the title screen the first hop also
// starts the game. A hop requested while the previous one is still animating
// is queued and fires as soon as the frog lands, so quick taps are never lost.
func (g *Game) Hop(d Dir) {
	switch g.state {
	case StateReady:
		g.Start()
	case StatePlaying:
	default:
		return
	}
	if g.frog.hopT > 0 {
		g.pending, g.hasPending = d, true
		return
	}
	g.doHop(d)
}

// doHop moves the frog immediately and starts the hop animation. Hops that
// would leave the board just turn the frog to face that way.
func (g *Game) doHop(d Dir) {
	f := &g.frog
	f.facing = d
	nx, nr := f.x, f.row
	switch d {
	case DirUp:
		nr++
	case DirDown:
		nr--
	case DirLeft:
		nx--
	case DirRight:
		nx++
	}
	if nr < startRow || nr > homeRow || nx < 0.5 || nx > cols-0.5 {
		g.needRender(RenderMove)
		return
	}
	// Land rows snap back to the column grid, so a frog that rode a log ashore
	// is tidy again. The river stays continuous and the home row has its own
	// bay snapping.
	if nr <= medianRow {
		nx = clamp(float32(math.Floor(float64(nx)))+0.5, 0.5, cols-0.5)
	}

	f.hopDX = nx - f.x
	f.hopDRow = float32(nr - f.row)
	f.hopT = hopDuration
	f.x, f.row = nx, nr
	g.play(SoundHop)

	if nr > g.highestRow {
		g.highestRow = nr
		g.score += 10
		g.needRender(RenderFull)
	}
	if nr == homeRow {
		g.arriveHome()
	}
}

// bayAt returns the index of the home bay under column x, or -1 for the bank.
func bayAt(x float32) int {
	for i, c := range homeCenters {
		if abs(x-c) <= bayTolerance {
			return i
		}
	}
	return -1
}

// arriveHome resolves a hop onto the top row: a splat against the bank or a
// filled bay, a meal for a crocodile, or a safe landing that scores and may
// complete the level.
func (g *Game) arriveHome() {
	f := &g.frog
	bay := bayAt(f.x)
	if bay < 0 || g.homes[bay] {
		g.die(DeathSquash)
		return
	}
	if g.visitor.kind == VisitorCroc && g.visitor.bay == bay {
		g.die(DeathSquash)
		return
	}

	// Snap into the bay, keeping the hop animation continuous.
	f.hopDX += homeCenters[bay] - f.x
	f.x = homeCenters[bay]

	g.homes[bay] = true
	g.score += 50 + 10*int(g.timeLeft)
	if g.visitor.kind == VisitorFly && g.visitor.bay == bay {
		g.score += 200
		g.visitor = bayVisitor{next: 4 + rand.Float32()*6}
		g.play(SoundFly)
	}
	g.checkExtraLife()

	if g.allHome() {
		g.score += 1000
		g.state = StateLevelClear
		g.stateT = 0
		g.needRender(RenderFull)
		g.play(SoundLevelUp)
		return
	}
	g.play(SoundHome)
	g.spawnFrog()
}

func (g *Game) allHome() bool {
	for _, h := range g.homes {
		if !h {
			return false
		}
	}
	return true
}

// checkExtraLife awards a frog every extraLifeEvery points.
func (g *Game) checkExtraLife() {
	for g.score >= g.nextExtra {
		g.lives++
		g.nextExtra += extraLifeEvery
		g.needRender(RenderFull)
		g.play(SoundExtraLife)
	}
}

// die starts the death animation; afterDeath decides what happens next.
func (g *Game) die(cause DeathCause) {
	g.state = StateDying
	g.stateT = 0
	g.deathCause = cause
	g.hasPending = false
	g.needRender(RenderFull)
	if cause == DeathDrown {
		g.play(SoundSplash)
	} else {
		g.play(SoundSquash)
	}
}

// afterDeath spends a life and either respawns or ends the game.
func (g *Game) afterDeath() {
	g.lives--
	if g.lives <= 0 {
		g.lives = 0
		g.state = StateGameOver
		g.needRender(RenderFull)
		g.play(SoundGameOver)
		return
	}
	g.spawnFrog()
	g.state = StatePlaying
}

// nextLevel follows the level-cleared celebration.
func (g *Game) nextLevel() {
	if g.level >= maxLevel {
		g.state = StateWon
		g.needRender(RenderFull)
		g.play(SoundWin)
		return
	}
	g.level++
	g.setupLevel()
	g.state = StatePlaying
}

// Tick advances the game by dt seconds and reports how much redrawing the
// frame needs. Traffic keeps flowing on the title and death screens so the
// board always feels alive; only Paused, Game Over and Won freeze it.
func (g *Game) Tick(dt float32) Render {
	switch g.state {
	case StatePlaying:
		g.advanceMovers(dt)
		g.updateFrog(dt)
		if g.state == StatePlaying { // updateFrog may have killed the frog
			g.updateVisitor(dt)
			g.updateTimer(dt)
		}
		g.needRender(RenderMove)
	case StateDying:
		g.advanceMovers(dt)
		g.animateHop(dt)
		g.stateT += dt
		if g.stateT >= dyingDuration {
			g.afterDeath()
		}
		g.needRender(RenderMove)
	case StateLevelClear:
		g.advanceMovers(dt)
		g.stateT += dt
		if g.stateT >= clearDuration {
			g.nextLevel()
		}
		g.needRender(RenderMove)
	case StateReady:
		g.advanceMovers(dt)
		g.needRender(RenderMove)
	}
	r := g.render
	g.render = RenderNone
	return r
}

func (g *Game) advanceMovers(dt float32) {
	for _, m := range g.movers {
		m.advance(dt)
	}
}

// animateHop runs down the hop animation clock.
func (g *Game) animateHop(dt float32) {
	if g.frog.hopT > 0 {
		g.frog.hopT -= dt
		if g.frog.hopT < 0 {
			g.frog.hopT = 0
		}
	}
}

// updateFrog animates the hop, fires any queued hop, then judges the frog's
// cell: squashed on the road, carried or drowned on the river.
func (g *Game) updateFrog(dt float32) {
	g.animateHop(dt)
	if g.frog.hopT == 0 && g.hasPending {
		g.hasPending = false
		g.doHop(g.pending)
		if g.state != StatePlaying {
			return
		}
	}

	f := &g.frog
	switch {
	case isRoad(f.row):
		if g.hitByVehicle() {
			g.die(DeathSquash)
		}
	case isRiver(f.row):
		p := g.platformUnder()
		if p == nil {
			g.die(DeathDrown)
			return
		}
		f.x += p.speed * dt
		if f.x < -0.2 || f.x > cols+0.2 {
			g.die(DeathDrown) // swept off the edge of the world
		}
	}
}

func isRoad(row int) bool  { return row > startRow && row < medianRow }
func isRiver(row int) bool { return row > medianRow && row < homeRow }

// hitByVehicle tests the frog's (slightly forgiving) box against the vehicles
// in its lane.
func (g *Game) hitByVehicle() bool {
	f := &g.frog
	for _, m := range g.movers {
		if m.row != f.row || !m.kind.IsVehicle() {
			continue
		}
		if f.x+frogHalfW > m.x+0.1 && f.x-frogHalfW < m.x+m.w-0.1 {
			return true
		}
	}
	return false
}

// platformUnder returns the log or surfaced turtle group beneath the frog's
// centre, or nil if it is treading water.
func (g *Game) platformUnder() *Mover {
	f := &g.frog
	for _, m := range g.movers {
		if m.row != f.row || m.kind.IsVehicle() || !m.supports() {
			continue
		}
		if f.x >= m.x-rideSlack && f.x <= m.x+m.w+rideSlack {
			return m
		}
	}
	return nil
}

// updateVisitor runs the fly / crocodile that drops in on empty bays.
func (g *Game) updateVisitor(dt float32) {
	v := &g.visitor
	if v.kind != VisitorNone {
		v.t -= dt
		if v.t <= 0 {
			*v = bayVisitor{next: 4 + rand.Float32()*6}
			g.needRender(RenderFull)
		}
		return
	}
	v.next -= dt
	if v.next > 0 {
		return
	}
	var empty []int
	for i, h := range g.homes {
		if !h {
			empty = append(empty, i)
		}
	}
	if len(empty) == 0 {
		v.next = 3
		return
	}
	v.bay = empty[rand.Intn(len(empty))]
	v.t = 6
	v.kind = VisitorFly
	if g.level >= 3 && rand.Float32() < 0.4 {
		v.kind = VisitorCroc
	}
	g.needRender(RenderFull)
}

// updateTimer counts down the frog's clock and kills it at zero.
func (g *Game) updateTimer(dt float32) {
	g.timeLeft -= dt
	if g.timeLeft <= 5 && !g.warned {
		g.warned = true
		g.play(SoundTimeWarning)
	}
	if g.timeLeft <= 0 {
		g.timeLeft = 0
		g.die(DeathTime)
	}
}

// FrogVisual returns where to draw the frog: its centre column and its row as a
// float, both eased back along the hop so the move reads as a jump, plus a
// 0..1 arc height that peaks mid-hop.
func (g *Game) FrogVisual() (x, row, arc float32) {
	f := &g.frog
	frac := f.hopT / hopDuration
	x = f.x - f.hopDX*frac
	row = float32(f.row) - f.hopDRow*frac
	arc = float32(math.Sin(float64(frac) * math.Pi))
	return x, row, arc
}

// --- small helpers ---

func clamp(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}
