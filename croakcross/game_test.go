package croakcross

import (
	"testing"
)

// newTestGame returns a game in play with every lane emptied, so tests can
// place exactly the traffic they need.
func newTestGame() *Game {
	g := NewGame()
	g.Start()
	g.movers = g.movers[:0]
	g.visitor.next = 1e9 // no surprise visitors
	return g
}

// addMover drops a single mover into the game and returns it.
func addMover(g *Game, kind Kind, row int, x, w, speed float32) *Mover {
	m := &Mover{kind: kind, row: row, x: x, w: w, speed: speed, period: 40}
	g.movers = append(g.movers, m)
	return m
}

// tick advances the game in small steps for a total of secs seconds.
func tick(g *Game, secs float32) {
	for t := float32(0); t < secs; t += 1.0 / 60 {
		g.Tick(1.0 / 60)
	}
}

func TestNewGameStartsOnTitle(t *testing.T) {
	g := NewGame()
	if g.state != StateReady {
		t.Fatalf("state = %v, want StateReady", g.state)
	}
	if g.lives != startLives || g.level != 1 || g.score != 0 {
		t.Fatalf("fresh game has lives=%d level=%d score=%d", g.lives, g.level, g.score)
	}
	if len(g.movers) == 0 {
		t.Fatal("level 1 should have traffic")
	}
	if g.frog.row != startRow || g.frog.x != float32(cols)/2 {
		t.Fatalf("frog spawned at (%v,%d), want centre of the start row", g.frog.x, g.frog.row)
	}
}

func TestFirstHopStartsGameAndScores(t *testing.T) {
	g := NewGame()
	g.movers = g.movers[:0]
	g.Hop(DirUp)
	if g.state != StatePlaying {
		t.Fatalf("hop on the title screen should start play, state=%v", g.state)
	}
	if g.frog.row != 1 || g.score != 10 {
		t.Fatalf("after one hop row=%d score=%d, want 1 and 10", g.frog.row, g.score)
	}
}

func TestForwardHopsScoreOnce(t *testing.T) {
	g := newTestGame()
	g.Hop(DirUp)
	tick(g, hopDuration*2)
	g.Hop(DirDown)
	tick(g, hopDuration*2)
	g.Hop(DirUp)
	if g.score != 10 {
		t.Fatalf("re-reaching a row should not score again, score=%d", g.score)
	}
	if g.frog.facing != DirUp {
		t.Fatalf("frog should face the last hop direction")
	}
}

func TestHopsAreQueuedWhileAnimating(t *testing.T) {
	g := newTestGame()
	g.Hop(DirUp)
	g.Hop(DirUp) // still mid-hop: queued, not dropped
	if g.frog.row != 1 {
		t.Fatalf("second hop should wait, row=%d", g.frog.row)
	}
	tick(g, hopDuration*2)
	if g.frog.row != 2 {
		t.Fatalf("queued hop should fire on landing, row=%d", g.frog.row)
	}
}

func TestEdgeHopsOnlyTurn(t *testing.T) {
	g := newTestGame()
	g.Hop(DirDown)
	if g.frog.row != startRow || g.frog.facing != DirDown {
		t.Fatalf("hop off the bottom should turn only: row=%d facing=%v", g.frog.row, g.frog.facing)
	}
	for i := 0; i < 20; i++ {
		g.Hop(DirLeft)
		tick(g, hopDuration*2)
	}
	if g.frog.x != 0.5 {
		t.Fatalf("frog left the board: x=%v", g.frog.x)
	}
}

func TestVehicleSquashesFrog(t *testing.T) {
	g := newTestGame()
	addMover(g, KindCar, 1, 6, 1, 0)
	g.Hop(DirUp)
	g.Tick(1.0 / 60)
	if g.state != StateDying || g.deathCause != DeathSquash {
		t.Fatalf("frog under a car should be squashed: state=%v cause=%v", g.state, g.deathCause)
	}
	tick(g, dyingDuration+0.1)
	if g.lives != startLives-1 || g.state != StatePlaying {
		t.Fatalf("after dying lives=%d state=%v", g.lives, g.state)
	}
	if g.frog.row != startRow {
		t.Fatal("frog should respawn on the start bank")
	}
}

func TestVehicleMissesFrogInOtherLane(t *testing.T) {
	g := newTestGame()
	addMover(g, KindCar, 2, 6, 1, 0)
	g.Hop(DirUp)
	tick(g, 0.5)
	if g.state != StatePlaying {
		t.Fatalf("car in another lane killed the frog")
	}
}

func TestLogCarriesFrog(t *testing.T) {
	g := newTestGame()
	g.frog.row, g.frog.x = 7, 6.5
	addMover(g, KindLog, 7, 5, 3, 2)
	tick(g, 1)
	if g.state != StatePlaying {
		t.Fatalf("frog on a log should survive, state=%v", g.state)
	}
	if g.frog.x < 8.3 || g.frog.x > 8.7 {
		t.Fatalf("frog should ride the log ~2 cells, x=%v", g.frog.x)
	}
}

func TestLogCarriesFrogOffEdge(t *testing.T) {
	g := newTestGame()
	g.frog.row, g.frog.x = 7, 12
	m := addMover(g, KindLog, 7, 11, 3, 3)
	m.period = 100 // keep it from wrapping back under the frog
	tick(g, 1)
	if g.state != StateDying || g.deathCause != DeathDrown {
		t.Fatalf("frog carried off-screen should drown: state=%v cause=%v", g.state, g.deathCause)
	}
}

func TestWaterDrownsFrog(t *testing.T) {
	g := newTestGame()
	g.frog.row = medianRow
	g.Hop(DirUp)
	g.Tick(1.0 / 60)
	if g.state != StateDying || g.deathCause != DeathDrown {
		t.Fatalf("frog in open water should drown: state=%v cause=%v", g.state, g.deathCause)
	}
}

func TestDivingTurtles(t *testing.T) {
	g := newTestGame()
	g.frog.row, g.frog.x = 10, 6.5
	m := addMover(g, KindTurtle, 10, 5, 3, -1)
	m.dives = true

	m.diveT = 1 // surfaced
	g.Tick(1.0 / 60)
	if g.state != StatePlaying {
		t.Fatal("surfaced turtles should carry the frog")
	}

	m.diveT = 5.5 // fully submerged
	g.Tick(1.0 / 60)
	if g.state != StateDying || g.deathCause != DeathDrown {
		t.Fatalf("submerged turtles should drown the frog: state=%v", g.state)
	}
}

func TestMoversWrapAround(t *testing.T) {
	g := NewGame()
	g.Start()
	tick(g, 60)
	for _, m := range g.movers {
		if m.x < -m.w-2 || m.x > cols+2 {
			t.Fatalf("mover drifted away: x=%v w=%v speed=%v", m.x, m.w, m.speed)
		}
	}
	if g.state != StateDying && g.state != StatePlaying {
		t.Fatalf("unexpected state %v", g.state)
	}
}

func TestHomeLandingScoresAndRespawns(t *testing.T) {
	g := newTestGame()
	g.frog.row, g.frog.x = 11, homeCenters[2]+0.3
	g.timeLeft = 20
	g.Hop(DirUp)
	if !g.homes[2] {
		t.Fatal("bay 2 should be filled")
	}
	want := 10 + 50 + 10*20
	if g.score != want {
		t.Fatalf("score=%d want %d", g.score, want)
	}
	if g.state != StatePlaying || g.frog.row != startRow || g.timeLeft != frogTime {
		t.Fatalf("a new frog should be on the bank with a full clock: state=%v row=%d time=%v",
			g.state, g.frog.row, g.timeLeft)
	}
}

func TestHomeBankSplatsFrog(t *testing.T) {
	g := newTestGame()
	g.frog.row, g.frog.x = 11, (homeCenters[0]+homeCenters[1])/2
	g.Hop(DirUp)
	if g.state != StateDying || g.deathCause != DeathSquash {
		t.Fatalf("landing between bays should splat: state=%v", g.state)
	}
}

func TestFilledBayKillsFrog(t *testing.T) {
	g := newTestGame()
	g.homes[1] = true
	g.frog.row, g.frog.x = 11, homeCenters[1]
	g.Hop(DirUp)
	if g.state != StateDying {
		t.Fatalf("landing in an occupied bay should kill: state=%v", g.state)
	}
}

func TestCrocodileEatsFrogAndFlyScores(t *testing.T) {
	g := newTestGame()
	g.visitor = bayVisitor{kind: VisitorCroc, bay: 3, t: 5}
	g.frog.row, g.frog.x = 11, homeCenters[3]
	g.Hop(DirUp)
	if g.state != StateDying {
		t.Fatalf("crocodile should eat the frog: state=%v", g.state)
	}

	g = newTestGame()
	g.visitor = bayVisitor{kind: VisitorFly, bay: 3, t: 5}
	g.frog.row, g.frog.x = 11, homeCenters[3]
	g.timeLeft = 10
	g.Hop(DirUp)
	if g.score != 10+50+100+200 {
		t.Fatalf("fly bonus missing: score=%d", g.score)
	}
	if g.visitor.kind != VisitorNone {
		t.Fatal("the fly should be eaten")
	}
}

func TestLevelClearAdvances(t *testing.T) {
	g := newTestGame()
	for i := 0; i < numHomes-1; i++ {
		g.homes[i] = true
	}
	g.frog.row, g.frog.x = 11, homeCenters[numHomes-1]
	before := g.score
	g.Hop(DirUp)
	if g.state != StateLevelClear {
		t.Fatalf("fifth frog home should clear the level: state=%v", g.state)
	}
	if g.score-before < 1000 {
		t.Fatalf("level bonus missing: gained %d", g.score-before)
	}
	tick(g, clearDuration+0.1)
	if g.level != 2 || g.state != StatePlaying {
		t.Fatalf("after the celebration level=%d state=%v", g.level, g.state)
	}
	for _, h := range g.homes {
		if h {
			t.Fatal("homes should be emptied for the new level")
		}
	}
	if len(g.movers) == 0 {
		t.Fatal("the new level should have traffic")
	}
}

func TestLevelsGetFaster(t *testing.T) {
	g := NewGame()
	speed1 := abs(g.movers[0].speed)
	g.level = 5
	g.setupLevel()
	if abs(g.movers[0].speed) <= speed1 {
		t.Fatalf("level 5 traffic should be faster than level 1: %v vs %v", g.movers[0].speed, speed1)
	}
}

func TestWinAfterLastLevel(t *testing.T) {
	g := newTestGame()
	g.level = maxLevel
	for i := range g.homes {
		g.homes[i] = true
	}
	g.state = StateLevelClear
	tick(g, clearDuration+0.1)
	if g.state != StateWon {
		t.Fatalf("clearing level %d should win: state=%v", maxLevel, g.state)
	}
}

func TestClockRunsOut(t *testing.T) {
	g := newTestGame()
	g.timeLeft = 0.5
	tick(g, 1)
	if g.state != StateDying || g.deathCause != DeathTime {
		t.Fatalf("frog should die when the clock hits zero: state=%v cause=%v", g.state, g.deathCause)
	}
}

func TestGameOver(t *testing.T) {
	g := newTestGame()
	g.lives = 1
	g.die(DeathSquash)
	tick(g, dyingDuration+0.1)
	if g.state != StateGameOver || g.lives != 0 {
		t.Fatalf("state=%v lives=%d, want game over", g.state, g.lives)
	}
	g.Hop(DirUp)
	if g.frog.row != startRow {
		t.Fatal("hops must be ignored after game over")
	}
	g.Restart()
	if g.state != StatePlaying || g.lives != startLives || g.score != 0 {
		t.Fatalf("restart should reset: state=%v lives=%d score=%d", g.state, g.lives, g.score)
	}
}

func TestExtraLife(t *testing.T) {
	g := newTestGame()
	g.score = extraLifeEvery - 60
	g.frog.row, g.frog.x = 11, homeCenters[0]
	g.timeLeft = 1
	g.Hop(DirUp) // +10 hop, +50 home, +10 time
	if g.lives != startLives+1 {
		t.Fatalf("crossing %d points should award a life, lives=%d", extraLifeEvery, g.lives)
	}
	if g.nextExtra != 2*extraLifeEvery {
		t.Fatalf("next extra life at %d, want %d", g.nextExtra, 2*extraLifeEvery)
	}
}

func TestPauseFreezesEverything(t *testing.T) {
	g := newTestGame()
	m := addMover(g, KindCar, 3, 0, 1, 2)
	g.TogglePause()
	x, clock := m.x, g.timeLeft
	tick(g, 1)
	if m.x != x || g.timeLeft != clock {
		t.Fatal("paused game must not advance")
	}
	g.Hop(DirUp)
	if g.frog.row != startRow {
		t.Fatal("hops must be ignored while paused")
	}
	g.TogglePause()
	tick(g, 0.5)
	if m.x == x {
		t.Fatal("resumed game should move again")
	}
}

func TestTickReportsRenderLevel(t *testing.T) {
	g := newTestGame()
	if r := g.Tick(1.0 / 60); r != RenderFull {
		t.Fatalf("first frame should be a full render, got %v", r)
	}
	if r := g.Tick(1.0 / 60); r != RenderMove {
		t.Fatalf("steady play should only need a move, got %v", r)
	}
	g.TogglePause()
	g.Tick(1.0 / 60) // consumes the pause's full render
	if r := g.Tick(1.0 / 60); r != RenderNone {
		t.Fatalf("paused game should need no render, got %v", r)
	}
}

func TestSoundsAreEmitted(t *testing.T) {
	g := newTestGame()
	var heard []Sound
	g.onSound = func(s Sound) { heard = append(heard, s) }
	g.Hop(DirUp)
	if len(heard) != 1 || heard[0] != SoundHop {
		t.Fatalf("hop should play SoundHop, heard %v", heard)
	}
}

func TestFrogVisualEasesIntoHop(t *testing.T) {
	g := newTestGame()
	g.Hop(DirUp)
	x, row, arc := g.FrogVisual()
	if x != g.frog.x || row != 0 || arc > 0.01 {
		t.Fatalf("at hop start the frog should still appear at its old cell: x=%v row=%v arc=%v", x, row, arc)
	}
	tick(g, hopDuration/2)
	_, row, arc = g.FrogVisual()
	if row <= 0 || row >= 1 || arc < 0.5 {
		t.Fatalf("mid-hop the frog should be between rows and in the air: row=%v arc=%v", row, arc)
	}
	tick(g, hopDuration)
	_, row, _ = g.FrogVisual()
	if row != 1 {
		t.Fatalf("after the hop the frog should sit on row 1, row=%v", row)
	}
}

func TestBayAt(t *testing.T) {
	for i, c := range homeCenters {
		if bayAt(c) != i || bayAt(c+0.5) != i || bayAt(c-0.5) != i {
			t.Fatalf("bayAt around centre %v should be %d", c, i)
		}
	}
	if bayAt(2.25) != -1 || bayAt(0) != -1 || bayAt(12.5) != -1 {
		t.Fatal("bank positions should not match a bay")
	}
}
