package moria

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

const (
	tileSize              = 16
	viewWidth             = 256
	viewHeight            = 240
	worldColumns          = 90
	worldWidth            = worldColumns * tileSize
	playerWidth           = 12
	playerHeight          = 16
	enemyWidth            = 14
	enemyHeight           = 12
	startX                = 32
	startY                = 192
	startLives            = 3
	gravity               = 900
	jumpVelocity          = 350
	walkSpeed     float32 = 88
	runSpeed      float32 = 142
	deathDuration         = 0.65
)

type State uint8

const (
	StateReady State = iota
	StatePlaying
	StatePaused
	StateDying
	StateGameOver
	StateWon
)

type Sound uint8

const (
	SoundJump Sound = iota
	SoundCrystal
	SoundStomp
	SoundHit
	SoundGoal
)

type Input struct {
	Left  bool
	Right bool
	Jump  bool
	Run   bool
}

type cell struct {
	x int
	y int
}

type tileKind uint8

const (
	tileStone tileKind = iota
	tileBlock
	tileCracked
	tileRune
)

type crystal struct {
	x         float32
	y         float32
	collected bool
}

type enemy struct {
	x     float32
	y     float32
	minX  float32
	maxX  float32
	speed float32
	alive bool
}

type player struct {
	x            float32
	y            float32
	prevY        float32
	vx           float32
	vy           float32
	onGround     bool
	facing       int
	jumpHeld     bool
	invulnerable float32
}

type Game struct {
	state        State
	player       player
	solids       map[cell]tileKind
	crystals     []crystal
	enemies      []enemy
	goalX        float32
	cameraX      float32
	lives        int
	score        int
	crystalCount int
	stateTime    float32
	animTime     float32
	onSound      func(Sound)
}

func NewGame() *Game {
	g := &Game{}
	g.Restart()
	g.state = StateReady
	return g
}

func (g *Game) Start() {
	if g.state == StateReady {
		g.state = StatePlaying
	}
}

func (g *Game) Pause() {
	if g.state == StatePlaying {
		g.state = StatePaused
	}
}

func (g *Game) TogglePause() {
	switch g.state {
	case StatePlaying:
		g.state = StatePaused
	case StatePaused:
		g.state = StatePlaying
	}
}

func (g *Game) Restart() {
	g.lives = startLives
	g.score = 0
	g.crystalCount = 0
	g.stateTime = 0
	g.cameraX = 0
	g.buildLevel()
	g.spawnPlayer()
	g.state = StatePlaying
}

func (g *Game) buildLevel() {
	g.solids = make(map[cell]tileKind)
	for _, span := range [][2]int{{0, 24}, {27, 43}, {46, 66}, {69, worldColumns}} {
		for x := span[0]; x < span[1]; x++ {
			g.solids[cell{x, 13}] = tileStone
			g.solids[cell{x, 14}] = tileStone
		}
	}
	for _, platform := range []struct {
		start int
		end   int
		row   int
	}{
		{8, 12, 9},
		{30, 35, 8},
		{49, 53, 10},
		{60, 64, 7},
	} {
		for x := platform.start; x < platform.end; x++ {
			g.solids[cell{x, platform.row}] = tileStone
		}
	}
	for _, pos := range []cell{{12, 9}, {38, 9}, {56, 8}} {
		g.solids[pos] = tileRune
	}
	for _, pos := range []cell{{37, 9}, {39, 9}, {40, 9}, {69, 8}, {70, 8}} {
		g.solids[pos] = tileCracked
	}
	g.crystals = []crystal{
		{x: 6*tileSize + 4, y: 11*tileSize + 2},
		{x: 9*tileSize + 4, y: 8*tileSize + 2},
		{x: 17*tileSize + 4, y: 11*tileSize + 2},
		{x: 24*tileSize + 4, y: 10*tileSize + 2},
		{x: 32*tileSize + 4, y: 7*tileSize + 2},
		{x: 40*tileSize + 4, y: 11*tileSize + 2},
		{x: 51*tileSize + 4, y: 9*tileSize + 2},
		{x: 62*tileSize + 4, y: 6*tileSize + 2},
		{x: 73*tileSize + 4, y: 11*tileSize + 2},
		{x: 80*tileSize + 4, y: 11*tileSize + 2},
	}
	g.enemies = []enemy{
		{x: 19 * tileSize, y: 196, minX: 17 * tileSize, maxX: 22 * tileSize, speed: 28, alive: true},
		{x: 35 * tileSize, y: 196, minX: 33 * tileSize, maxX: 40 * tileSize, speed: -34, alive: true},
		{x: 54 * tileSize, y: 196, minX: 50 * tileSize, maxX: 60 * tileSize, speed: 30, alive: true},
		{x: 76 * tileSize, y: 196, minX: 73 * tileSize, maxX: 81 * tileSize, speed: -36, alive: true},
	}
	g.goalX = 84 * tileSize
}

func (g *Game) spawnPlayer() {
	g.player = player{x: startX, y: startY, onGround: true, facing: 1, invulnerable: 1}
}

func (g *Game) Update(dt float32, input Input) {
	if dt <= 0 {
		return
	}
	if dt > 0.05 {
		dt = 0.05
	}
	g.animTime += dt

	if g.state == StateReady {
		if input.Left || input.Right || input.Jump {
			g.Start()
		}
	}
	if g.state == StateGameOver || g.state == StateWon {
		if input.Jump && !g.player.jumpHeld {
			g.Restart()
		}
		g.player.jumpHeld = input.Jump
		return
	}
	if g.state == StatePaused || g.state == StateReady {
		g.player.jumpHeld = input.Jump
		return
	}
	if g.state == StateDying {
		g.stateTime += dt
		if g.stateTime >= deathDuration {
			if g.lives == 0 {
				g.state = StateGameOver
			} else {
				g.spawnPlayer()
				g.state = StatePlaying
			}
		}
		g.player.jumpHeld = input.Jump
		return
	}

	g.updatePlayer(dt, input)
	g.updateEnemies(dt)
	g.collectCrystals()
	g.checkEnemyCollisions()
	if g.player.y > viewHeight+tileSize {
		g.player.invulnerable = 0
		g.takeHit()
	}
	if g.state == StatePlaying && overlaps(g.player.x, g.player.y, playerWidth, playerHeight, g.goalX, 160, 28, 48) {
		g.state = StateWon
		g.play(SoundGoal)
	}
	g.cameraX = clamp(g.player.x-viewWidth*0.34, 0, worldWidth-viewWidth)
}

func (g *Game) updatePlayer(dt float32, input Input) {
	p := &g.player
	p.invulnerable = max(0, p.invulnerable-dt)
	direction := float32(0)
	if input.Left {
		direction--
	}
	if input.Right {
		direction++
	}
	if direction != 0 {
		p.facing = int(direction)
		p.vx += direction * 480 * dt
		speed := walkSpeed
		if input.Run {
			speed = runSpeed
		}
		p.vx = clamp(p.vx, -speed, speed)
	} else {
		p.vx *= max(0, 1-8*dt)
		if abs(p.vx) < 3 {
			p.vx = 0
		}
	}

	if input.Jump && !p.jumpHeld && p.onGround {
		p.vy = -jumpVelocity
		p.onGround = false
		g.play(SoundJump)
	}
	if !input.Jump && p.vy < -120 {
		p.vy = -120
	}
	p.jumpHeld = input.Jump

	p.x += p.vx * dt
	g.resolveHorizontal()
	previousY := p.y
	p.prevY = previousY
	p.vy = min(p.vy+gravity*dt, 520)
	p.y += p.vy * dt
	p.onGround = false
	g.resolveVertical(previousY)
}

func (g *Game) resolveHorizontal() {
	p := &g.player
	minX := int(math.Floor(float64(p.x / tileSize)))
	maxX := int(math.Floor(float64((p.x + playerWidth - 0.01) / tileSize)))
	minY := int(math.Floor(float64(p.y / tileSize)))
	maxY := int(math.Floor(float64((p.y + playerHeight - 0.01) / tileSize)))
	for tx := minX; tx <= maxX; tx++ {
		for ty := minY; ty <= maxY; ty++ {
			if _, solid := g.solids[cell{tx, ty}]; !solid {
				continue
			}
			if p.vx > 0 {
				p.x = float32(tx*tileSize - playerWidth)
			} else if p.vx < 0 {
				p.x = float32((tx + 1) * tileSize)
			}
			p.vx = 0
		}
	}
	p.x = clamp(p.x, 0, worldWidth-playerWidth)
}

func (g *Game) resolveVertical(previousY float32) {
	p := &g.player
	minX := int(math.Floor(float64(p.x / tileSize)))
	maxX := int(math.Floor(float64((p.x + playerWidth - 0.01) / tileSize)))
	minY := int(math.Floor(float64(min(previousY, p.y) / tileSize)))
	maxY := int(math.Floor(float64((max(previousY, p.y) + playerHeight - 0.01) / tileSize)))
	for tx := minX; tx <= maxX; tx++ {
		for ty := minY; ty <= maxY; ty++ {
			pos := cell{tx, ty}
			kind, solid := g.solids[pos]
			if !solid {
				continue
			}
			tileTop := float32(ty * tileSize)
			tileBottom := float32((ty + 1) * tileSize)
			if p.vy > 0 && previousY+playerHeight <= tileTop && p.y+playerHeight >= tileTop {
				p.y = tileTop - playerHeight
				p.vy = 0
				p.onGround = true
			} else if p.vy < 0 && previousY >= tileBottom && p.y <= tileBottom {
				switch kind {
				case tileCracked:
					delete(g.solids, pos)
					g.score += 25
					g.play(SoundStomp)
					continue
				case tileRune:
					g.solids[pos] = tileBlock
					g.crystals = append(g.crystals, crystal{
						x: float32(tx*tileSize + 4),
						y: float32(ty*tileSize - 12),
					})
					g.play(SoundStomp)
				}
				p.y = tileBottom
				p.vy = 0
			}
		}
	}
}

func (g *Game) updateEnemies(dt float32) {
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive {
			continue
		}
		e.x += e.speed * dt
		if e.x < e.minX {
			e.x = e.minX
			e.speed = abs(e.speed)
		} else if e.x > e.maxX {
			e.x = e.maxX
			e.speed = -abs(e.speed)
		}
	}
}

func (g *Game) collectCrystals() {
	for i := range g.crystals {
		c := &g.crystals[i]
		if c.collected || !overlaps(g.player.x, g.player.y, playerWidth, playerHeight, c.x, c.y, 8, 12) {
			continue
		}
		c.collected = true
		g.crystalCount++
		g.score += 100
		g.play(SoundCrystal)
	}
}

func (g *Game) checkEnemyCollisions() {
	p := &g.player
	for i := range g.enemies {
		e := &g.enemies[i]
		if !e.alive || !overlaps(p.x, p.y, playerWidth, playerHeight, e.x, e.y, enemyWidth, enemyHeight) {
			continue
		}
		if p.vy > 0 && p.prevY+playerHeight <= e.y+4 {
			e.alive = false
			p.vy = -230
			p.onGround = false
			g.score += 150
			g.play(SoundStomp)
			continue
		}
		if p.invulnerable <= 0 {
			g.takeHit()
		}
	}
}

func (g *Game) takeHit() {
	if g.state != StatePlaying || g.player.invulnerable > 0 {
		return
	}
	g.lives--
	g.state = StateDying
	g.stateTime = 0
	g.play(SoundHit)
}

func (g *Game) play(sound Sound) {
	if g.onSound != nil {
		g.onSound(sound)
	}
}

func overlaps(ax, ay, aw, ah, bx, by, bw, bh float32) bool {
	return ax < bx+bw && ax+aw > bx && ay < by+bh && ay+ah > by
}

func clamp(v, low, high float32) float32 {
	return min(max(v, low), high)
}

func abs(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func (g *Game) renderImage(width, height int) image.Image {
	if width < 1 || height < 1 {
		return image.NewRGBA(image.Rect(0, 0, 1, 1))
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	scale := min(float32(width)/viewWidth, float32(height)/viewHeight)
	offsetX := (float32(width) - viewWidth*scale) / 2
	offsetY := (float32(height) - viewHeight*scale) / 2

	fill := func(x, y, w, h float32, c color.Color) {
		left := int(math.Floor(float64(offsetX + x*scale)))
		top := int(math.Floor(float64(offsetY + y*scale)))
		right := int(math.Ceil(float64(offsetX + (x+w)*scale)))
		bottom := int(math.Ceil(float64(offsetY + (y+h)*scale)))
		rect := image.Rect(left, top, right, bottom).Intersect(img.Bounds())
		if rect.Empty() {
			return
		}
		draw.Draw(img, rect, &image.Uniform{C: c}, image.Point{}, draw.Over)
	}

	fillScreen := func(x, y, w, h float32, c color.Color) {
		fill(x, y, w, h, c)
	}

	fillScreen(0, 0, viewWidth, viewHeight, colCave)
	fillScreen(0, 0, viewWidth, 28, colCeiling)
	for x := -float32(math.Mod(float64(g.cameraX), 48)); x < viewWidth; x += 48 {
		fill(x, 12, 32, 4, colRockLight)
		fill(x+7, 16, 20, 5, colRock)
		fill(x+12, 21, 10, 5, colRockDark)
	}
	fillScreen(0, 0, viewWidth, 2, colGlow)

	firstColumn := int(g.cameraX/tileSize) - 1
	lastColumn := firstColumn + viewWidth/tileSize + 3
	for pos, kind := range g.solids {
		if pos.x < firstColumn || pos.x > lastColumn {
			continue
		}
		x := float32(pos.x*tileSize) - g.cameraX
		y := float32(pos.y * tileSize)
		base, highlight := colStone, colStoneLight
		switch kind {
		case tileBlock:
			base, highlight = colBlock, colBlockLight
		case tileCracked:
			base, highlight = colCracked, colCrackedLight
		case tileRune:
			base, highlight = colRune, colRuneLight
		}
		fill(x, y, tileSize, tileSize, base)
		fill(x, y, tileSize, 3, highlight)
		fill(x+2, y+6, 4, 2, colStoneShade)
		fill(x+10, y+11, 4, 2, colStoneShade)
		if kind == tileCracked {
			fill(x+7, y+3, 1, 9, colStoneShade)
			fill(x+4, y+8, 7, 1, colStoneShade)
		} else if kind == tileRune {
			fill(x+7, y+5, 2, 6, colCrystal)
			fill(x+5, y+7, 6, 2, colCrystal)
		}
	}

	for _, c := range g.crystals {
		if c.collected {
			continue
		}
		x := c.x - g.cameraX
		fill(x+3, c.y, 3, 2, colCrystalLight)
		fill(x+1, c.y+2, 7, 4, colCrystal)
		fill(x+3, c.y+6, 4, 4, colCrystalDark)
		fill(x+3, c.y+2, 2, 3, colCrystalLight)
	}

	for _, e := range g.enemies {
		if !e.alive {
			continue
		}
		x := e.x - g.cameraX
		fill(x+2, e.y+5, 10, 6, colSlug)
		fill(x+4, e.y+2, 7, 5, colSlugLight)
		fill(x+5, e.y+5, 2, 2, colEye)
		fill(x+9, e.y+5, 2, 2, colEye)
		fill(x+6, e.y+6, 1, 1, colInk)
		fill(x+10, e.y+6, 1, 1, colInk)
		fill(x+1, e.y+10, 4, 2, colSlugDark)
		fill(x+9, e.y+10, 4, 2, colSlugDark)
	}

	gateX := g.goalX - g.cameraX
	fill(gateX+2, 164, 24, 4, colGateDark)
	fill(gateX+4, 166, 4, 42, colGate)
	fill(gateX+20, 166, 4, 42, colGate)
	fill(gateX+6, 164, 16, 4, colGate)
	fill(gateX+9, 172, 10, 28, colGateGlow)
	fill(gateX+12, 178, 4, 14, colCrystalLight)
	fill(gateX, 204, 28, 4, colGateBase)

	if g.state != StateDying || int(g.animTime*12)%2 == 0 {
		g.drawPlayer(fill)
	}
	return img
}

func (g *Game) drawPlayer(fill func(float32, float32, float32, float32, color.Color)) {
	p := g.player
	x, y := p.x-g.cameraX, p.y
	if p.invulnerable > 0 && int(g.animTime*18)%2 == 0 {
		return
	}
	fill(x+3, y, 7, 3, colHelmet)
	fill(x+5, y-2, 4, 2, colLamp)
	fill(x+4, y+3, 7, 4, colSkin)
	fill(x+9, y+4, 1, 1, colInk)
	fill(x+5, y+6, 6, 2, colBeard)
	fill(x+2, y+8, 9, 5, colTunic)
	fill(x+1, y+11, 11, 2, colTunicLight)
	fill(x+2, y+13, 4, 3, colBoot)
	fill(x+8, y+13, 4, 3, colBoot)
	if p.facing > 0 {
		fill(x+11, y+8, 2, 5, colPick)
	} else {
		fill(x-1, y+8, 2, 5, colPick)
	}
}

var (
	colCave         = color.NRGBA{R: 23, G: 21, B: 40, A: 255}
	colCeiling      = color.NRGBA{R: 37, G: 32, B: 56, A: 255}
	colRockLight    = color.NRGBA{R: 91, G: 74, B: 111, A: 255}
	colRock         = color.NRGBA{R: 65, G: 54, B: 81, A: 255}
	colRockDark     = color.NRGBA{R: 43, G: 38, B: 59, A: 255}
	colGlow         = color.NRGBA{R: 130, G: 94, B: 168, A: 255}
	colStone        = color.NRGBA{R: 62, G: 55, B: 78, A: 255}
	colStoneLight   = color.NRGBA{R: 105, G: 85, B: 116, A: 255}
	colStoneShade   = color.NRGBA{R: 47, G: 43, B: 65, A: 255}
	colBlock        = color.NRGBA{R: 101, G: 77, B: 102, A: 255}
	colBlockLight   = color.NRGBA{R: 159, G: 118, B: 137, A: 255}
	colCracked      = color.NRGBA{R: 82, G: 65, B: 90, A: 255}
	colCrackedLight = color.NRGBA{R: 139, G: 104, B: 142, A: 255}
	colRune         = color.NRGBA{R: 51, G: 75, B: 88, A: 255}
	colRuneLight    = color.NRGBA{R: 101, G: 188, B: 179, A: 255}
	colCrystal      = color.NRGBA{R: 82, G: 220, B: 197, A: 255}
	colCrystalDark  = color.NRGBA{R: 48, G: 151, B: 155, A: 255}
	colCrystalLight = color.NRGBA{R: 196, G: 255, B: 236, A: 255}
	colSlug         = color.NRGBA{R: 138, G: 98, B: 172, A: 255}
	colSlugLight    = color.NRGBA{R: 192, G: 148, B: 214, A: 255}
	colSlugDark     = color.NRGBA{R: 83, G: 58, B: 112, A: 255}
	colEye          = color.NRGBA{R: 255, G: 236, B: 153, A: 255}
	colInk          = color.NRGBA{R: 32, G: 26, B: 44, A: 255}
	colHelmet       = color.NRGBA{R: 219, G: 125, B: 57, A: 255}
	colLamp         = color.NRGBA{R: 255, G: 206, B: 104, A: 255}
	colSkin         = color.NRGBA{R: 235, G: 183, B: 145, A: 255}
	colBeard        = color.NRGBA{R: 134, G: 78, B: 61, A: 255}
	colTunic        = color.NRGBA{R: 55, G: 150, B: 141, A: 255}
	colTunicLight   = color.NRGBA{R: 81, G: 193, B: 166, A: 255}
	colBoot         = color.NRGBA{R: 63, G: 53, B: 78, A: 255}
	colPick         = color.NRGBA{R: 223, G: 194, B: 133, A: 255}
	colGate         = color.NRGBA{R: 68, G: 204, B: 184, A: 255}
	colGateDark     = color.NRGBA{R: 35, G: 83, B: 89, A: 255}
	colGateGlow     = color.NRGBA{R: 41, G: 98, B: 111, A: 255}
	colGateBase     = color.NRGBA{R: 186, G: 146, B: 208, A: 255}
)
