package starbreaker

import (
	"math"
	"testing"
)

// newPlayingGame returns a started game on a fixed field, with no asteroids so
// tests control collisions explicitly.
func newPlayingGame() *Game {
	g := NewGame()
	g.Resize(800, 800) // first Resize initialises the field
	g.StartGame()
	g.asteroids = g.asteroids[:0]
	return g
}

func approx(a, b float32) bool { return math.Abs(float64(a-b)) < 1e-3 }

func TestVecRotation(t *testing.T) {
	// Rotating "up" by 90° clockwise (screen space, +y down) points right.
	r := (vec{0, -1}).rot(math.Pi / 2)
	if !approx(r.x, 1) || !approx(r.y, 0) {
		t.Fatalf("rot 90° = %+v, want {1,0}", r)
	}
}

func TestWrapAxis(t *testing.T) {
	g := &Game{w: 100, h: 100}
	cases := []struct{ in, want float32 }{
		{-10, 90}, {110, 10}, {50, 50}, {0, 0},
	}
	for _, c := range cases {
		if got := g.wrapAxis(c.in, 100); !approx(got, c.want) {
			t.Errorf("wrapAxis(%v) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestToroidalDistance(t *testing.T) {
	g := &Game{w: 100, h: 100}
	// Points at x=2 and x=98 are 4 apart across the seam, not 96.
	if d2 := g.dist2(vec{2, 50}, vec{98, 50}); !approx(d2, 16) {
		t.Fatalf("toroidal dist2 = %v, want 16", d2)
	}
}

func TestStartGameState(t *testing.T) {
	g := newPlayingGame()
	g.nextWave() // refill rocks we cleared in the helper
	if g.state != StatePlaying {
		t.Fatalf("state = %v, want StatePlaying", g.state)
	}
	if !g.ship.alive {
		t.Fatal("ship should be alive after StartGame")
	}
	if g.lives != startLives {
		t.Fatalf("lives = %d, want %d", g.lives, startLives)
	}
}

func TestFireRespectsCooldownAndMax(t *testing.T) {
	g := newPlayingGame()
	g.ship.invuln = 0
	g.Fire()
	if g.liveBullets() != 1 {
		t.Fatalf("after one Fire, bullets = %d, want 1", g.liveBullets())
	}
	g.Fire() // still on cooldown
	if g.liveBullets() != 1 {
		t.Fatalf("cooldown ignored: bullets = %d, want 1", g.liveBullets())
	}
	// Clear cooldown and fill to the cap.
	for i := 0; i < maxBullets+3; i++ {
		g.fireCool = 0
		g.Fire()
	}
	if g.liveBullets() != maxBullets {
		t.Fatalf("bullets = %d, want capped at %d", g.liveBullets(), maxBullets)
	}
}

func TestBulletExpires(t *testing.T) {
	g := newPlayingGame()
	g.fireCool = 0
	g.Fire()
	if g.liveBullets() != 1 {
		t.Fatal("expected one bullet")
	}
	// Advance past the bullet lifetime.
	for t0 := float32(0); t0 < bulletLife+0.1; t0 += 0.05 {
		g.stepBullets(0.05)
	}
	if len(g.bullets) != 0 {
		t.Fatalf("bullet should have expired, have %d", len(g.bullets))
	}
}

func TestAsteroidSplitAndScore(t *testing.T) {
	g := newPlayingGame()
	g.asteroids = g.asteroids[:0]
	g.spawnAsteroid(sizeLarge, vec{400, 400})

	g.destroyAsteroid(0)
	if len(g.asteroids) != 2 {
		t.Fatalf("large rock should split into 2, got %d", len(g.asteroids))
	}
	for _, a := range g.asteroids {
		if a.size != sizeMedium {
			t.Fatalf("child size = %d, want medium", a.size)
		}
	}

	// A small rock just shatters.
	g.asteroids = g.asteroids[:0]
	g.spawnAsteroid(sizeSmall, vec{400, 400})
	g.destroyAsteroid(0)
	if len(g.asteroids) != 0 {
		t.Fatalf("small rock should vanish, got %d", len(g.asteroids))
	}
}

func TestBulletHitScores(t *testing.T) {
	g := newPlayingGame()
	g.asteroids = g.asteroids[:0]
	g.spawnAsteroid(sizeSmall, vec{400, 400})
	g.bullets = append(g.bullets, &Bullet{pos: vec{400, 400}, life: 1})
	g.bulletHits()
	if g.score != asteroidScore[sizeSmall] {
		t.Fatalf("score = %d, want %d", g.score, asteroidScore[sizeSmall])
	}
	if len(g.asteroids) != 0 || len(g.bullets) != 0 {
		t.Fatalf("hit should consume both bullet and rock; rocks=%d bullets=%d", len(g.asteroids), len(g.bullets))
	}
}

func TestSaucerBulletDoesNotScore(t *testing.T) {
	g := newPlayingGame()
	g.asteroids = g.asteroids[:0]
	g.spawnAsteroid(sizeSmall, vec{400, 400})
	g.bullets = append(g.bullets, &Bullet{pos: vec{400, 400}, life: 1, fromSaucer: true})
	g.bulletHits()
	if g.score != 0 {
		t.Fatalf("saucer shot should not score, got %d", g.score)
	}
	if len(g.asteroids) != 0 {
		t.Fatal("saucer shot should still destroy the rock")
	}
}

func TestWaveAdvancesWhenCleared(t *testing.T) {
	g := newPlayingGame()
	g.asteroids = g.asteroids[:0]
	wave := g.wave
	g.Tick(0.016) // empty field → next wave
	if g.wave != wave+1 {
		t.Fatalf("wave = %d, want %d", g.wave, wave+1)
	}
	if len(g.asteroids) == 0 {
		t.Fatal("new wave should spawn rocks")
	}
}

func TestShipDeathAndRespawn(t *testing.T) {
	g := newPlayingGame()
	g.ship.invuln = 0
	lives := g.lives
	g.killShip()
	if g.ship.alive {
		t.Fatal("ship should be dead")
	}
	if g.lives != lives-1 {
		t.Fatalf("lives = %d, want %d", g.lives, lives-1)
	}
	// Wait out the respawn countdown.
	for t0 := float32(0); t0 < respawnWait+0.1; t0 += 0.05 {
		g.stepShip(0.05)
	}
	if !g.ship.alive {
		t.Fatal("ship should respawn while lives remain")
	}
	if g.ship.invuln <= 0 {
		t.Fatal("respawned ship should be briefly invulnerable")
	}
}

func TestGameOverWhenOutOfLives(t *testing.T) {
	g := newPlayingGame()
	g.ship.invuln = 0
	g.lives = 1
	g.killShip() // drops to 0 lives
	for t0 := float32(0); t0 < respawnWait+0.1; t0 += 0.05 {
		g.stepShip(0.05)
	}
	if g.state != StateGameOver {
		t.Fatalf("state = %v, want StateGameOver", g.state)
	}
}

func TestInvulnerableShipSurvives(t *testing.T) {
	g := newPlayingGame()
	g.asteroids = g.asteroids[:0]
	g.spawnAsteroid(sizeLarge, g.ship.pos) // rock right on top of the ship
	g.ship.invuln = invulnTime
	g.shipHits()
	if !g.ship.alive {
		t.Fatal("invulnerable ship should survive a rock")
	}
}

func TestExtraLifeAtThreshold(t *testing.T) {
	g := newPlayingGame()
	lives := g.lives
	g.addScore(extraLifeEvery)
	if g.lives != lives+1 {
		t.Fatalf("lives = %d, want %d after crossing the bonus threshold", g.lives, lives+1)
	}
}

func TestHyperspaceMovesShip(t *testing.T) {
	g := newPlayingGame()
	before := g.ship.pos
	moved := false
	for i := 0; i < 20 && !moved; i++ {
		g.Hyperspace()
		if g.ship.pos != before {
			moved = true
		}
	}
	if !moved {
		t.Fatal("hyperspace should relocate the ship")
	}
	if g.ship.vel != (vec{}) {
		t.Fatal("hyperspace should zero the ship's momentum")
	}
}

func TestPauseToggle(t *testing.T) {
	g := newPlayingGame()
	g.TogglePause()
	if g.state != StatePaused {
		t.Fatalf("state = %v, want StatePaused", g.state)
	}
	g.TogglePause()
	if g.state != StatePlaying {
		t.Fatalf("state = %v, want StatePlaying", g.state)
	}
}

func TestWrapOffsetsNearCorner(t *testing.T) {
	g := &Game{w: 100, h: 100}
	// Dead centre: only the identity offset.
	if got := len(g.wrapOffsets(vec{50, 50}, 5)); got != 1 {
		t.Fatalf("centre offsets = %d, want 1", got)
	}
	// Hard in a corner: identity + x + y + xy = 4 copies.
	if got := len(g.wrapOffsets(vec{1, 1}, 5)); got != 4 {
		t.Fatalf("corner offsets = %d, want 4", got)
	}
}
