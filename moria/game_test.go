package moria

import (
	"image"
	"testing"
)

func playingGame() *Game {
	game := NewGame()
	game.Start()
	game.player.invulnerable = 0
	for i := range game.enemies {
		game.enemies[i].alive = false
	}
	return game
}

func advance(game *Game, duration float32, input Input) {
	for elapsed := float32(0); elapsed < duration; elapsed += 1.0 / 60 {
		game.Update(1.0/60, input)
	}
}

func TestNewGameStartsReadyWithCavernContent(t *testing.T) {
	game := NewGame()
	if game.state != StateReady || game.lives != startLives {
		t.Fatalf("new game state=%v lives=%d", game.state, game.lives)
	}
	if len(game.solids) == 0 || len(game.crystals) < 5 || len(game.enemies) == 0 {
		t.Fatal("cavern should contain platforms, crystals, and enemies")
	}
}

func TestMovementJumpAndRun(t *testing.T) {
	game := playingGame()
	startX, startY := game.player.x, game.player.y
	for i := 0; i < 12; i++ {
		game.Update(1.0/60, Input{Right: true, Jump: true, Run: true})
	}
	if game.player.x <= startX+10 {
		t.Fatalf("running should move the player forward: x=%v", game.player.x)
	}
	if game.player.y >= startY-20 {
		t.Fatalf("holding jump should lift the player: y=%v", game.player.y)
	}
}

func TestRuneBlockReleasesCrystalAndCrackedBlockBreaks(t *testing.T) {
	game := playingGame()
	rune := cell{3, 10}
	game.solids[rune] = tileRune
	game.player.x = float32(rune.x*tileSize + 2)
	game.player.y = float32((rune.y + 1) * tileSize)
	game.player.onGround = true
	game.Update(1.0/60, Input{Jump: true})
	if game.solids[rune] != tileBlock || len(game.crystals) != 11 {
		t.Fatal("hitting a rune block should turn it to stone and release a crystal")
	}

	cracked := cell{4, 10}
	game.solids[cracked] = tileCracked
	game.player.x = float32(cracked.x*tileSize + 2)
	game.player.y = float32((cracked.y + 1) * tileSize)
	game.player.onGround = true
	game.player.vy = 0
	game.player.jumpHeld = false
	game.Update(1.0/60, Input{Jump: true})
	if _, exists := game.solids[cracked]; exists {
		t.Fatal("a cracked block should break when struck from below")
	}
}

func TestCrystalCollectionAndGoal(t *testing.T) {
	game := playingGame()
	game.player.x = game.crystals[0].x
	game.player.y = game.crystals[0].y
	game.collectCrystals()
	if game.crystalCount != 1 || game.score != 100 || !game.crystals[0].collected {
		t.Fatal("touching a crystal should collect it and award points")
	}

	game.player.x = game.goalX
	game.player.y = 192
	game.Update(1.0/60, Input{})
	if game.state != StateWon {
		t.Fatal("reaching the ancient gate should win the game")
	}
}

func TestStompingAndTakingDamage(t *testing.T) {
	game := NewGame()
	game.Start()
	e := &game.enemies[0]
	game.player.x = e.x
	game.player.y = e.y - playerHeight - 2
	game.player.vy = 220
	game.player.onGround = false
	game.player.invulnerable = 0
	game.Update(1.0/60, Input{})
	if e.alive || game.score != 150 {
		t.Fatal("landing on a cave crawler should defeat it and award points")
	}

	game = NewGame()
	game.Start()
	e = &game.enemies[0]
	game.player.x = e.x
	game.player.y = 192
	game.player.onGround = true
	game.player.invulnerable = 0
	game.Update(1.0/60, Input{})
	if game.state != StateDying || game.lives != startLives-1 {
		t.Fatal("touching an enemy from the side should cost a life")
	}
}

func TestFallingRespawnsAndPauseFreezes(t *testing.T) {
	game := NewGame()
	game.Start()
	game.player.y = viewHeight + tileSize
	game.Update(1.0/60, Input{})
	if game.state != StateDying || game.lives != startLives-1 {
		t.Fatal("falling into a pit should cost a life")
	}
	advance(game, deathDuration, Input{})
	if game.state != StatePlaying || game.player.x != startX {
		t.Fatal("the player should respawn at the cave entrance")
	}

	game.Pause()
	x := game.player.x
	game.Update(0.05, Input{Right: true, Jump: true})
	if game.state != StatePaused || game.player.x != x {
		t.Fatal("pausing should freeze movement")
	}
}

func TestGameOverRestartAndRenderSize(t *testing.T) {
	game := playingGame()
	game.lives = 1
	game.player.invulnerable = 0
	game.takeHit()
	advance(game, deathDuration, Input{})
	if game.state != StateGameOver {
		t.Fatal("losing the final life should end the game")
	}
	game.Update(1.0/60, Input{Jump: true})
	if game.state != StatePlaying || game.lives != startLives {
		t.Fatal("jumping after game over should restart")
	}

	img := game.renderImage(512, 480)
	if img.Bounds() != image.Rect(0, 0, 512, 480) {
		t.Fatalf("render bounds = %v", img.Bounds())
	}
}
