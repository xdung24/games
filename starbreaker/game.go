package starbreaker

import (
	"math"
	"math/rand"
)

// GameState models the high-level phase of play.
type GameState int

const (
	StateAttract  GameState = iota // title screen; asteroids drift behind the banner
	StatePlaying                   // ship in play
	StatePaused                    // frozen by the player
	StateGameOver                  // out of lives
)

// Tuning constants that don't depend on the board size.
const (
	startLives = 3

	rotSpeed     = 3.2  // ship turn rate, radians/sec
	shipDrag     = 0.55 // velocity damping per second (space is *almost* frictionless)
	maxBullets   = 5    // bullets on screen at once
	bulletLife   = 0.72 // seconds a bullet flies before fizzling
	fireCooldown = 0.16 // minimum seconds between shots

	invulnTime  = 2.2 // spawn-protection after respawning, seconds
	respawnWait = 1.6 // pause between death and the next ship, seconds

	extraLifeEvery = 10000 // a bonus ship at each multiple of this score
)

// Asteroid size tiers. Larger rocks are worth less and split into two of the
// next size down; the smallest simply shatters.
const (
	sizeSmall = iota
	sizeMedium
	sizeLarge
	numSizes
)

// asteroidScore rewards the smaller, harder-to-hit rocks more.
var asteroidScore = [numSizes]int{100, 50, 20}

// Sound names an audible game event. The model only emits these through a
// callback so it stays free of any audio dependency (and the tests stay silent).
type Sound int

const (
	SoundFire Sound = iota
	SoundThrust
	SoundBangLarge
	SoundBangMedium
	SoundBangSmall
	SoundShipDeath
	SoundExtraLife
	SoundSaucer
	SoundSaucerFire
	SoundNewWave
)

// vec is a tiny 2-D vector used throughout the model so the geometry stays
// readable. Everything is float32 to match Fyne's canvas coordinates.
type vec struct{ x, y float32 }

func (a vec) add(b vec) vec       { return vec{a.x + b.x, a.y + b.y} }
func (a vec) sub(b vec) vec       { return vec{a.x - b.x, a.y - b.y} }
func (a vec) scale(s float32) vec { return vec{a.x * s, a.y * s} }
func (a vec) len() float32        { return float32(math.Hypot(float64(a.x), float64(a.y))) }

// rot returns a rotated by angle radians (screen space, +y down).
func (a vec) rot(angle float32) vec {
	s, c := float32(math.Sin(float64(angle))), float32(math.Cos(float64(angle)))
	return vec{a.x*c - a.y*s, a.x*s + a.y*c}
}

// Ship is the player's craft: a momentum-carrying triangle that wraps the screen.
type Ship struct {
	pos, vel vec
	angle    float32 // radians; 0 points straight up
	thrust   bool    // engine firing this frame (drives the flame + particles)
	alive    bool
	invuln   float32 // remaining spawn-protection, seconds
	respawn  float32 // countdown to the next ship after a death, seconds
}

// Asteroid is a drifting, spinning rock. shape holds unit-radius vertices so the
// jagged silhouette is fixed for the rock's life; render scales and rotates them.
type Asteroid struct {
	pos, vel vec
	angle    float32 // current rotation
	spin     float32 // rotation rate, radians/sec
	radius   float32
	size     int
	shape    []vec // unit polygon (|v| ~ 1), generated once
}

// Bullet is a short-lived shot. Saucer shots are tracked too so they can hit the
// player but never trigger the player's own scoring.
type Bullet struct {
	pos, vel   vec
	life       float32
	fromSaucer bool
}

// Particle is a fading line fragment thrown out by explosions and the thruster.
type Particle struct {
	pos, vel      vec
	life, maxLife float32
}

// Saucer is the occasional hunter that crosses the field and shoots at the ship.
type Saucer struct {
	pos, vel  vec
	alive     bool
	big       bool    // big saucers are easy points; small ones aim true
	fireTimer float32 // countdown to the next shot
}

// Game holds all mutable play state. Every field is touched only from the UI
// goroutine (the ticker schedules updates via fyne.Do), so no locking is needed.
type Game struct {
	w, h        float32 // logical play-field size, kept equal to the widget size
	initialized bool

	ship      Ship
	asteroids []*Asteroid
	bullets   []*Bullet
	particles []*Particle
	saucer    Saucer

	score     int
	lives     int
	wave      int
	nextExtra int // score threshold for the next bonus ship
	state     GameState

	leftHeld, rightHeld, thrustHeld bool
	fireHeld                        bool    // auto-fire while the fire key is down
	fireCool                        float32 // remaining shot cooldown
	saucerTimer                     float32 // countdown to the next saucer
	thrustNoise                     float32 // throttles thruster sound retriggers
	dirty                           bool    // force a redraw next frame (state changes)

	onSound func(Sound) // optional sink for sound events; nil = silent
}

// play emits a sound event if a sink is wired up.
func (g *Game) play(s Sound) {
	if g.onSound != nil {
		g.onSound(s)
	}
}

// NewGame returns a game sitting on the attract screen at a default size.
func NewGame() *Game {
	g := &Game{w: 800, h: 800, lives: startLives, state: StateAttract}
	return g
}

// Resize keeps the logical field matched to the widget. The first call seeds the
// attract-mode rocks; later calls rescale everything so a window resize is smooth.
func (g *Game) Resize(w, h float32) {
	if w <= 0 || h <= 0 {
		return
	}
	g.dirty = true
	if !g.initialized {
		g.w, g.h = w, h
		g.initialized = true
		g.startAttract()
		return
	}
	sx, sy := w/g.w, h/g.h
	g.w, g.h = w, h
	scalePos := func(p vec) vec { return vec{p.x * sx, p.y * sy} }
	s := float32(math.Sqrt(float64(sx * sy)))

	g.ship.pos = scalePos(g.ship.pos)
	g.ship.vel = scalePos(g.ship.vel)
	for _, a := range g.asteroids {
		a.pos = scalePos(a.pos)
		a.vel = scalePos(a.vel)
		a.radius *= s
	}
	for _, b := range g.bullets {
		b.pos = scalePos(b.pos)
		b.vel = scalePos(b.vel)
	}
	for _, p := range g.particles {
		p.pos = scalePos(p.pos)
		p.vel = scalePos(p.vel)
	}
	if g.saucer.alive {
		g.saucer.pos = scalePos(g.saucer.pos)
		g.saucer.vel = scalePos(g.saucer.vel)
	}
}

// --- field-relative sizing helpers -----------------------------------------

func (g *Game) shipRadius() float32  { return g.h * 0.022 }
func (g *Game) bulletSpeed() float32 { return g.h * 0.95 }
func (g *Game) thrustAccel() float32 { return g.h * 0.95 }
func (g *Game) maxSpeed() float32    { return g.h * 0.62 }
func (g *Game) saucerSpeed() float32 { return g.h * 0.20 }
func (g *Game) saucerRadius(big bool) float32 {
	if big {
		return g.h * 0.030
	}
	return g.h * 0.020
}

// asteroidRadius returns the radius for a size tier.
func (g *Game) asteroidRadius(size int) float32 {
	switch size {
	case sizeLarge:
		return g.h * 0.058
	case sizeMedium:
		return g.h * 0.035
	default:
		return g.h * 0.020
	}
}

// --- setup ------------------------------------------------------------------

// startAttract fills the field with slow rocks behind the title screen.
func (g *Game) startAttract() {
	g.state = StateAttract
	g.asteroids = g.asteroids[:0]
	g.bullets = g.bullets[:0]
	g.particles = g.particles[:0]
	g.saucer = Saucer{}
	for i := 0; i < 6; i++ {
		g.spawnAsteroid(sizeLarge, g.randomEdgePos())
	}
	g.dirty = true
}

// StartGame begins a fresh game from the attract screen (or a game-over).
func (g *Game) StartGame() {
	g.score = 0
	g.lives = startLives
	g.wave = 0
	g.nextExtra = extraLifeEvery
	g.bullets = g.bullets[:0]
	g.particles = g.particles[:0]
	g.saucer = Saucer{}
	g.saucerTimer = g.randRange(12, 22)
	g.state = StatePlaying
	g.spawnShip()
	g.nextWave()
	g.dirty = true
}

// nextWave clears the board and spawns the next, larger field of rocks.
func (g *Game) nextWave() {
	g.wave++
	g.asteroids = g.asteroids[:0]
	count := 4 + g.wave
	if count > 11 {
		count = 11
	}
	for i := 0; i < count; i++ {
		g.spawnAsteroid(sizeLarge, g.randomEdgePos())
	}
	if g.wave > 1 {
		g.play(SoundNewWave)
	}
}

// spawnShip parks a fresh, briefly-invulnerable ship at the centre, stationary.
func (g *Game) spawnShip() {
	g.ship = Ship{
		pos:    vec{g.w / 2, g.h / 2},
		angle:  0,
		alive:  true,
		invuln: invulnTime,
	}
}

// spawnAsteroid adds a rock of the given size at pos with a random drift, spin
// and jagged outline.
func (g *Game) spawnAsteroid(size int, pos vec) {
	a := &Asteroid{
		pos:    pos,
		size:   size,
		radius: g.asteroidRadius(size),
		spin:   g.randRange(-1.4, 1.4),
		shape:  randomAsteroidShape(),
	}
	// Smaller rocks move faster, giving the late game its frantic feel.
	speed := g.h * (0.05 + 0.03*float32(numSizes-size)) * g.randRange(0.6, 1.4)
	dir := g.randRange(0, 2*math.Pi)
	a.vel = vec{speed * float32(math.Cos(float64(dir))), speed * float32(math.Sin(float64(dir)))}
	g.asteroids = append(g.asteroids, a)
}

// randomAsteroidShape builds a jagged unit polygon with 9–12 vertices.
func randomAsteroidShape() []vec {
	n := 9 + rand.Intn(4)
	pts := make([]vec, n)
	for i := 0; i < n; i++ {
		ang := 2 * math.Pi * float64(i) / float64(n)
		r := 0.72 + rand.Float64()*0.45 // jitter the radius for a craggy edge
		pts[i] = vec{float32(math.Cos(ang) * r), float32(math.Sin(ang) * r)}
	}
	return pts
}

// randomEdgePos returns a point on a random edge of the field, used so new rocks
// drift in rather than appearing on top of the ship.
func (g *Game) randomEdgePos() vec {
	if rand.Intn(2) == 0 {
		return vec{g.randRange(0, g.w), boolPick(rand.Intn(2) == 0, 0, g.h)}
	}
	return vec{boolPick(rand.Intn(2) == 0, 0, g.w), g.randRange(0, g.h)}
}

// --- input ------------------------------------------------------------------

// Fire launches a bullet from the ship's nose if a slot and the cooldown allow.
func (g *Game) Fire() {
	if g.state != StatePlaying || !g.ship.alive {
		return
	}
	if g.fireCool > 0 || g.liveBullets() >= maxBullets {
		return
	}
	dir := vec{0, -1}.rot(g.ship.angle)
	nose := g.ship.pos.add(dir.scale(g.shipRadius()))
	g.bullets = append(g.bullets, &Bullet{
		pos:  nose,
		vel:  g.ship.vel.add(dir.scale(g.bulletSpeed())),
		life: bulletLife,
	})
	g.fireCool = fireCooldown
	g.play(SoundFire)
}

// Hyperspace jumps the ship to a random spot — an escape hatch that drops all
// momentum and protection, so it can also drop you onto a rock.
func (g *Game) Hyperspace() {
	if g.state != StatePlaying || !g.ship.alive {
		return
	}
	g.ship.pos = vec{g.randRange(0, g.w), g.randRange(0, g.h)}
	g.ship.vel = vec{}
	g.ship.invuln = 0.4
}

// TogglePause flips between Playing and Paused.
func (g *Game) TogglePause() {
	switch g.state {
	case StatePlaying:
		g.state = StatePaused
	case StatePaused:
		g.state = StatePlaying
	}
	g.dirty = true
}

// Pause freezes play if a game is in progress (used when the app backgrounds).
func (g *Game) Pause() {
	if g.state == StatePlaying {
		g.state = StatePaused
		g.dirty = true
	}
}

// Activate is the primary action: start from the attract/over screens, or fire
// while playing. The board maps Space and taps onto it.
func (g *Game) Activate() {
	switch g.state {
	case StateAttract, StateGameOver:
		g.StartGame()
	case StatePlaying:
		g.Fire()
	}
}

// --- simulation -------------------------------------------------------------

// Tick advances the game by dt seconds and reports whether the frame needs a
// redraw. Play and attract animate every frame; paused/over screens redraw only
// on the frame their state changed.
func (g *Game) Tick(dt float32) bool {
	moving := g.state == StatePlaying || g.state == StateAttract
	switch g.state {
	case StateAttract:
		g.stepAsteroids(dt)
		g.stepParticles(dt)
	case StatePlaying:
		if g.fireHeld {
			g.Fire() // cooldown-limited, so holding the key auto-fires
		}
		g.stepShip(dt)
		g.stepAsteroids(dt)
		g.stepBullets(dt)
		g.stepParticles(dt)
		g.stepSaucer(dt)
		g.collide()
		if len(g.asteroids) == 0 {
			g.nextWave()
		}
	}
	d := g.dirty
	g.dirty = false
	return moving || d
}

// stepShip applies rotation, thrust and drag, then integrates and wraps the ship.
func (g *Game) stepShip(dt float32) {
	if g.fireCool > 0 {
		g.fireCool -= dt
	}
	if !g.ship.alive {
		// Counting down to the next respawn (or game over if out of lives).
		g.ship.respawn -= dt
		if g.ship.respawn <= 0 {
			if g.lives > 0 {
				g.spawnShip()
			} else {
				g.state = StateGameOver
				g.dirty = true
			}
		}
		return
	}
	if g.ship.invuln > 0 {
		g.ship.invuln -= dt
	}
	if g.leftHeld {
		g.ship.angle -= rotSpeed * dt
	}
	if g.rightHeld {
		g.ship.angle += rotSpeed * dt
	}

	g.ship.thrust = g.thrustHeld
	if g.thrustHeld {
		dir := vec{0, -1}.rot(g.ship.angle)
		g.ship.vel = g.ship.vel.add(dir.scale(g.thrustAccel() * dt))
		if sp := g.ship.vel.len(); sp > g.maxSpeed() {
			g.ship.vel = g.ship.vel.scale(g.maxSpeed() / sp)
		}
		g.emitThrust(dir)
		// Retrigger the engine rumble periodically rather than every frame.
		g.thrustNoise -= dt
		if g.thrustNoise <= 0 {
			g.play(SoundThrust)
			g.thrustNoise = 0.12
		}
	}
	// Gentle drag so a drifting ship eventually slows — kinder than pure inertia.
	g.ship.vel = g.ship.vel.scale(float32(math.Max(0, float64(1-shipDrag*dt))))
	g.ship.pos = g.wrap(g.ship.pos.add(g.ship.vel.scale(dt)))
}

// emitThrust sprays a couple of short-lived particles out of the ship's tail.
func (g *Game) emitThrust(dir vec) {
	tail := g.ship.pos.sub(dir.scale(g.shipRadius()))
	for i := 0; i < 2; i++ {
		spread := g.randRange(-0.5, 0.5)
		v := dir.scale(-g.h * 0.25).rot(spread).add(g.ship.vel)
		g.particles = append(g.particles, &Particle{
			pos: tail, vel: v, life: 0.35, maxLife: 0.35,
		})
	}
}

// stepAsteroids drifts and spins every rock, wrapping at the edges.
func (g *Game) stepAsteroids(dt float32) {
	for _, a := range g.asteroids {
		a.pos = g.wrap(a.pos.add(a.vel.scale(dt)))
		a.angle += a.spin * dt
	}
}

// stepBullets ages bullets and drops the expired ones; live bullets wrap.
func (g *Game) stepBullets(dt float32) {
	live := g.bullets[:0]
	for _, b := range g.bullets {
		b.life -= dt
		if b.life <= 0 {
			continue
		}
		b.pos = g.wrap(b.pos.add(b.vel.scale(dt)))
		live = append(live, b)
	}
	g.bullets = live
}

// stepParticles ages and moves debris, dropping the dead.
func (g *Game) stepParticles(dt float32) {
	live := g.particles[:0]
	for _, p := range g.particles {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.pos = p.pos.add(p.vel.scale(dt))
		live = append(live, p)
	}
	g.particles = live
}

// stepSaucer schedules, moves and fires the hunter craft.
func (g *Game) stepSaucer(dt float32) {
	if !g.saucer.alive {
		g.saucerTimer -= dt
		if g.saucerTimer <= 0 {
			g.spawnSaucer()
		}
		return
	}
	s := &g.saucer
	s.pos.x += s.vel.x * dt
	s.pos.y = g.wrapAxis(s.pos.y+s.vel.y*dt, g.h)
	// Occasionally jink vertically so it isn't a sitting duck.
	if rand.Float32() < dt*0.8 {
		s.vel.y = g.randRange(-1, 1) * g.saucerSpeed() * 0.6
	}
	// Off the far side: the saucer leaves and the next one is rescheduled.
	if (s.vel.x > 0 && s.pos.x > g.w+g.saucerRadius(s.big)) ||
		(s.vel.x < 0 && s.pos.x < -g.saucerRadius(s.big)) {
		g.saucer.alive = false
		g.saucerTimer = g.randRange(14, 26)
		return
	}
	s.fireTimer -= dt
	if s.fireTimer <= 0 && g.ship.alive {
		g.saucerFire()
		s.fireTimer = g.randRange(0.8, 1.6)
	}
}

// spawnSaucer launches a saucer from a random side at a random height.
func (g *Game) spawnSaucer() {
	big := rand.Intn(3) != 0 // small saucers are rarer and deadlier
	fromLeft := rand.Intn(2) == 0
	s := Saucer{alive: true, big: big, fireTimer: g.randRange(0.6, 1.4)}
	if fromLeft {
		s.pos = vec{-g.saucerRadius(big), g.randRange(g.h*0.2, g.h*0.8)}
		s.vel = vec{g.saucerSpeed(), 0}
	} else {
		s.pos = vec{g.w + g.saucerRadius(big), g.randRange(g.h*0.2, g.h*0.8)}
		s.vel = vec{-g.saucerSpeed(), 0}
	}
	g.saucer = s
	g.play(SoundSaucer)
}

// saucerFire shoots toward the ship; big saucers spray wildly, small ones aim.
func (g *Game) saucerFire() {
	aim := g.ship.pos.sub(g.saucer.pos)
	ang := float32(math.Atan2(float64(aim.y), float64(aim.x)))
	if g.saucer.big {
		ang += g.randRange(-0.5, 0.5)
	} else {
		ang += g.randRange(-0.08, 0.08)
	}
	dir := vec{float32(math.Cos(float64(ang))), float32(math.Sin(float64(ang)))}
	g.bullets = append(g.bullets, &Bullet{
		pos:        g.saucer.pos.add(dir.scale(g.saucerRadius(g.saucer.big))),
		vel:        dir.scale(g.bulletSpeed() * 0.8),
		life:       bulletLife * 1.3,
		fromSaucer: true,
	})
	g.play(SoundSaucerFire)
}

// --- collisions -------------------------------------------------------------

// collide resolves every interaction for the frame: shots vs rocks/saucer, and
// the ship vs rocks/saucer/saucer-fire.
func (g *Game) collide() {
	g.bulletHits()
	g.shipHits()
}

// bulletHits checks player shots against rocks and the saucer, and saucer shots
// against rocks (so the saucer can clear its own path, classic-style).
func (g *Game) bulletHits() {
	kept := g.bullets[:0]
	for _, b := range g.bullets {
		hit := false
		for i, a := range g.asteroids {
			if g.dist2(b.pos, a.pos) <= a.radius*a.radius {
				g.destroyAsteroid(i)
				if !b.fromSaucer {
					g.addScore(asteroidScore[a.size])
				}
				hit = true
				break
			}
		}
		if hit {
			continue
		}
		// A player shot can also down the saucer.
		if !b.fromSaucer && g.saucer.alive {
			if g.dist2(b.pos, g.saucer.pos) <= sq(g.saucerRadius(g.saucer.big)) {
				g.killSaucer(true)
				continue
			}
		}
		kept = append(kept, b)
	}
	g.bullets = kept
}

// shipHits ends the ship's run if (while vulnerable) it touches a rock, the
// saucer, or a saucer bullet.
func (g *Game) shipHits() {
	if !g.ship.alive || g.ship.invuln > 0 {
		return
	}
	r := g.shipRadius() * 0.7 // forgiving hull, slightly tighter than the visual
	for i, a := range g.asteroids {
		if g.dist2(g.ship.pos, a.pos) <= sq(a.radius+r) {
			g.destroyAsteroid(i)
			g.killShip()
			return
		}
	}
	if g.saucer.alive && g.dist2(g.ship.pos, g.saucer.pos) <= sq(g.saucerRadius(g.saucer.big)+r) {
		g.killSaucer(false)
		g.killShip()
		return
	}
	kept := g.bullets[:0]
	for _, b := range g.bullets {
		if b.fromSaucer && g.dist2(g.ship.pos, b.pos) <= sq(r) {
			g.killShip()
			continue
		}
		kept = append(kept, b)
	}
	g.bullets = kept
}

// destroyAsteroid removes rock i, spawning its two children (unless it's small)
// and a burst of debris and the right-sized bang.
func (g *Game) destroyAsteroid(i int) {
	a := g.asteroids[i]
	g.asteroids = append(g.asteroids[:i], g.asteroids[i+1:]...)
	g.explode(a.pos, a.radius)
	switch a.size {
	case sizeLarge:
		g.play(SoundBangLarge)
	case sizeMedium:
		g.play(SoundBangMedium)
	default:
		g.play(SoundBangSmall)
	}
	if a.size == sizeSmall {
		return
	}
	for c := 0; c < 2; c++ {
		child := &Asteroid{
			size:   a.size - 1,
			radius: g.asteroidRadius(a.size - 1),
			pos:    a.pos,
			spin:   g.randRange(-2, 2),
			shape:  randomAsteroidShape(),
		}
		speed := a.vel.len()*1.1 + g.h*0.04
		dir := g.randRange(0, 2*math.Pi)
		child.vel = vec{speed * float32(math.Cos(float64(dir))), speed * float32(math.Sin(float64(dir)))}
		g.asteroids = append(g.asteroids, child)
	}
}

// killSaucer removes the saucer, scoring it only when the player shot it down.
func (g *Game) killSaucer(scored bool) {
	g.explode(g.saucer.pos, g.saucerRadius(g.saucer.big))
	if scored {
		if g.saucer.big {
			g.addScore(200)
		} else {
			g.addScore(1000)
		}
	}
	g.play(SoundBangLarge)
	g.saucer.alive = false
	g.saucerTimer = g.randRange(14, 26)
}

// killShip blows up the ship, spends a life and arms the respawn countdown.
func (g *Game) killShip() {
	g.explode(g.ship.pos, g.shipRadius()*1.5)
	g.ship.alive = false
	g.ship.thrust = false
	g.ship.respawn = respawnWait
	g.lives--
	g.play(SoundShipDeath)
	g.dirty = true
}

// explode throws a ring of fading line-fragments out from p.
func (g *Game) explode(p vec, radius float32) {
	n := 10 + rand.Intn(8)
	for i := 0; i < n; i++ {
		ang := g.randRange(0, 2*math.Pi)
		spd := radius * g.randRange(3, 8)
		life := g.randRange(0.4, 0.9)
		g.particles = append(g.particles, &Particle{
			pos:     p,
			vel:     vec{spd * float32(math.Cos(float64(ang))), spd * float32(math.Sin(float64(ang)))},
			life:    life,
			maxLife: life,
		})
	}
}

// addScore credits points and grants a bonus ship at each extra-life threshold.
func (g *Game) addScore(n int) {
	g.score += n
	for g.score >= g.nextExtra {
		g.lives++
		g.nextExtra += extraLifeEvery
		g.play(SoundExtraLife)
	}
}

// --- helpers ----------------------------------------------------------------

// wrap brings a position back onto the field on both axes (toroidal world).
func (g *Game) wrap(p vec) vec {
	return vec{g.wrapAxis(p.x, g.w), g.wrapAxis(p.y, g.h)}
}

// wrapAxis wraps v into [0,size).
func (g *Game) wrapAxis(v, size float32) float32 {
	if size <= 0 {
		return v
	}
	for v < 0 {
		v += size
	}
	for v >= size {
		v -= size
	}
	return v
}

// liveBullets counts the player's shots currently in flight.
func (g *Game) liveBullets() int {
	n := 0
	for _, b := range g.bullets {
		if !b.fromSaucer {
			n++
		}
	}
	return n
}

// randRange returns a uniform float32 in [lo, hi).
func (g *Game) randRange(lo, hi float32) float32 { return lo + rand.Float32()*(hi-lo) }

func boolPick(cond bool, a, b float32) float32 {
	if cond {
		return a
	}
	return b
}

// dist2 is the squared distance between two field points on the toroidal world,
// taking the shorter way around each wrapped axis so edge-straddling objects
// collide exactly as they appear (the render draws their wrapped copies too).
func (g *Game) dist2(a, b vec) float32 {
	dx := wrapDelta(a.x-b.x, g.w)
	dy := wrapDelta(a.y-b.y, g.h)
	return dx*dx + dy*dy
}

// wrapDelta folds a coordinate difference into [-size/2, size/2].
func wrapDelta(d, size float32) float32 {
	if size <= 0 {
		return d
	}
	for d > size/2 {
		d -= size
	}
	for d < -size/2 {
		d += size
	}
	return d
}

func sq(v float32) float32 { return v * v }
