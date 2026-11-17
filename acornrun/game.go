package acornrun

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
	SoundAcorn
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
	tileGrass tileKind = iota
	tileWood
	tileCrackedWood
	tileAcornBlock
)

type acorn struct {
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
	state      State
	player     player
	solids     map[cell]tileKind
	acorns     []acorn
	beetles    []enemy
	goalX      float32
	cameraX    float32
	lives      int
	score      int
	acornCount int
	stateTime  float32
	animTime   float32
	onSound    func(Sound)
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
	g.acornCount = 0
	g.stateTime = 0
	g.cameraX = 0
	g.buildLevel()
	g.spawnPlayer()
	g.state = StatePlaying
}

func (g *Game) buildLevel() {
	g.solids = make(map[cell]tileKind)
	for _, span := range [][2]int{{0, 18}, {22, 38}, {42, 59}, {63, 78}, {82, worldColumns}} {
		for x := span[0]; x < span[1]; x++ {
			g.solids[cell{x, 13}] = tileGrass
			g.solids[cell{x, 14}] = tileGrass
		}
	}
	for _, platform := range []struct {
		start int
		end   int
		row   int
	}{
		{5, 9, 10},
		{12, 16, 8},
		{25, 30, 9},
		{34, 39, 7},
		{45, 50, 10},
		{55, 59, 8},
		{67, 72, 9},
		{74, 79, 6},
	} {
		for x := platform.start; x < platform.end; x++ {
			g.solids[cell{x, platform.row}] = tileGrass
		}
	}
	for _, pos := range []cell{{9, 9}, {29, 8}, {50, 9}, {72, 8}} {
		g.solids[pos] = tileAcornBlock
	}
	for _, pos := range []cell{{23, 9}, {24, 9}, {37, 7}, {38, 7}, {63, 9}, {64, 9}, {83, 8}} {
		g.solids[pos] = tileCrackedWood
	}
	g.acorns = []acorn{
		{x: 4*tileSize + 4, y: 11*tileSize + 2},
		{x: 7*tileSize + 4, y: 9*tileSize + 2},
		{x: 14*tileSize + 4, y: 7*tileSize + 2},
		{x: 20*tileSize + 4, y: 11*tileSize + 2},
		{x: 27*tileSize + 4, y: 7*tileSize + 2},
		{x: 33*tileSize + 4, y: 11*tileSize + 2},
		{x: 40*tileSize + 4, y: 11*tileSize + 2},
		{x: 47*tileSize + 4, y: 8*tileSize + 2},
		{x: 53*tileSize + 4, y: 11*tileSize + 2},
		{x: 58*tileSize + 4, y: 8*tileSize + 2},
		{x: 68*tileSize + 4, y: 7*tileSize + 2},
		{x: 76*tileSize + 4, y: 5*tileSize + 2},
		{x: 84*tileSize + 4, y: 11*tileSize + 2},
	}
	g.beetles = []enemy{
		{x: 14 * tileSize, y: 196, minX: 11 * tileSize, maxX: 17 * tileSize, speed: 28, alive: true},
		{x: 30 * tileSize, y: 196, minX: 25 * tileSize, maxX: 35 * tileSize, speed: -34, alive: true},
		{x: 49 * tileSize, y: 196, minX: 45 * tileSize, maxX: 56 * tileSize, speed: 30, alive: true},
		{x: 73 * tileSize, y: 196, minX: 68 * tileSize, maxX: 76 * tileSize, speed: -36, alive: true},
	}
	g.goalX = 86 * tileSize
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
	g.collectAcorns()
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
				case tileCrackedWood:
					delete(g.solids, pos)
					g.score += 25
					g.play(SoundStomp)
					continue
				case tileAcornBlock:
					g.solids[pos] = tileWood
					g.acorns = append(g.acorns, acorn{
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
	for i := range g.beetles {
		e := &g.beetles[i]
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

func (g *Game) collectAcorns() {
	for i := range g.acorns {
		c := &g.acorns[i]
		if c.collected || !overlaps(g.player.x, g.player.y, playerWidth, playerHeight, c.x, c.y, 8, 12) {
			continue
		}
		c.collected = true
		g.acornCount++
		g.score += 100
		g.play(SoundAcorn)
	}
}

func (g *Game) checkEnemyCollisions() {
	p := &g.player
	for i := range g.beetles {
		e := &g.beetles[i]
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

	fillScreen(0, 0, viewWidth, viewHeight, colForestSky)
	fillScreen(0, 0, viewWidth, 38, colCanopyDark)
	for x := -float32(math.Mod(float64(g.cameraX), 48)); x < viewWidth; x += 48 {
		fill(x, 14, 32, 10, colLeafDark)
		fill(x+7, 20, 20, 13, colLeaf)
		fill(x+12, 27, 10, 12, colLeafLight)
	}
	for x := -float32(math.Mod(float64(g.cameraX*0.25), 72)); x < viewWidth; x += 72 {
		fill(x, 78, 42, 12, colHill)
		fill(x+12, 67, 24, 16, colHill)
	}
	fillScreen(0, 0, viewWidth, 2, colSunlight)

	firstColumn := int(g.cameraX/tileSize) - 1
	lastColumn := firstColumn + viewWidth/tileSize + 3
	for pos, kind := range g.solids {
		if pos.x < firstColumn || pos.x > lastColumn {
			continue
		}
		x := float32(pos.x*tileSize) - g.cameraX
		y := float32(pos.y * tileSize)
		base, highlight := colDirt, colGrass
		switch kind {
		case tileWood:
			base, highlight = colWood, colWoodLight
		case tileCrackedWood:
			base, highlight = colCrackedWood, colCrackedWoodLight
		case tileAcornBlock:
			base, highlight = colAcornBlock, colAcornBlockLight
		}
		fill(x, y, tileSize, tileSize, base)
		fill(x, y, tileSize, 3, highlight)
		if kind == tileGrass {
			fill(x+2, y+6, 4, 2, colDirtShade)
			fill(x+10, y+11, 4, 2, colDirtShade)
		} else if kind == tileCrackedWood {
			fill(x+7, y+3, 1, 9, colWoodShade)
			fill(x+4, y+8, 7, 1, colWoodShade)
		} else if kind == tileAcornBlock {
			fill(x+6, y+4, 4, 8, colAcorn)
			fill(x+4, y+7, 8, 2, colAcornLight)
		}
	}

	for _, c := range g.acorns {
		if c.collected {
			continue
		}
		x := c.x - g.cameraX
		fill(x+3, c.y, 3, 2, colAcornCap)
		fill(x+1, c.y+2, 7, 5, colAcorn)
		fill(x+3, c.y+7, 4, 3, colAcornDark)
		fill(x+2, c.y+2, 2, 3, colAcornLight)
	}

	for _, e := range g.beetles {
		if !e.alive {
			continue
		}
		x := e.x - g.cameraX
		fill(x+2, e.y+5, 10, 6, colBeetle)
		fill(x+4, e.y+2, 7, 5, colBeetleShell)
		fill(x+5, e.y+5, 2, 2, colEye)
		fill(x+9, e.y+5, 2, 2, colEye)
		fill(x+6, e.y+6, 1, 1, colInk)
		fill(x+10, e.y+6, 1, 1, colInk)
		fill(x+1, e.y+10, 4, 2, colBeetleDark)
		fill(x+9, e.y+10, 4, 2, colBeetleDark)
	}

	treeX := g.goalX - g.cameraX
	fill(treeX+4, 164, 20, 44, colBark)
	fill(treeX+1, 169, 5, 39, colBarkDark)
	fill(treeX+22, 169, 5, 39, colBarkDark)
	fill(treeX+8, 177, 12, 31, colDoorway)
	fill(treeX+3, 160, 22, 5, colLeaf)
	fill(treeX, 155, 28, 6, colLeafLight)
	fill(treeX+12, 185, 4, 5, colWindow)
	fill(treeX+10, 205, 8, 3, colBarkDark)

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
	fill(x+10, y+7, 4, 5, colTail)
	fill(x+11, y+5, 3, 3, colTailLight)
	fill(x+3, y+1, 7, 6, colFur)
	fill(x+4, y-1, 2, 3, colFurDark)
	fill(x+8, y, 2, 3, colFurDark)
	fill(x+4, y+5, 6, 3, colFurLight)
	if p.facing > 0 {
		fill(x+8, y+5, 1, 1, colInk)
	} else {
		fill(x+4, y+5, 1, 1, colInk)
	}
	fill(x+2, y+8, 9, 5, colVest)
	fill(x+1, y+10, 3, 4, colAcorn)
	fill(x+1, y+11, 11, 2, colVestLight)
	fill(x+2, y+13, 4, 3, colBoot)
	fill(x+8, y+13, 4, 3, colBoot)
}

var (
	colForestSky        = color.NRGBA{R: 168, G: 220, B: 205, A: 255}
	colCanopyDark       = color.NRGBA{R: 37, G: 93, B: 71, A: 255}
	colLeafDark         = color.NRGBA{R: 50, G: 117, B: 78, A: 255}
	colLeaf             = color.NRGBA{R: 69, G: 145, B: 83, A: 255}
	colLeafLight        = color.NRGBA{R: 119, G: 176, B: 93, A: 255}
	colHill             = color.NRGBA{R: 111, G: 168, B: 123, A: 255}
	colSunlight         = color.NRGBA{R: 244, G: 222, B: 147, A: 255}
	colDirt             = color.NRGBA{R: 129, G: 91, B: 57, A: 255}
	colGrass            = color.NRGBA{R: 88, G: 156, B: 77, A: 255}
	colDirtShade        = color.NRGBA{R: 98, G: 69, B: 47, A: 255}
	colWood             = color.NRGBA{R: 145, G: 94, B: 55, A: 255}
	colWoodLight        = color.NRGBA{R: 190, G: 132, B: 75, A: 255}
	colWoodShade        = color.NRGBA{R: 96, G: 59, B: 43, A: 255}
	colCrackedWood      = color.NRGBA{R: 107, G: 73, B: 51, A: 255}
	colCrackedWoodLight = color.NRGBA{R: 167, G: 116, B: 71, A: 255}
	colAcornBlock       = color.NRGBA{R: 179, G: 126, B: 62, A: 255}
	colAcornBlockLight  = color.NRGBA{R: 229, G: 180, B: 87, A: 255}
	colAcorn            = color.NRGBA{R: 175, G: 104, B: 47, A: 255}
	colAcornDark        = color.NRGBA{R: 122, G: 69, B: 43, A: 255}
	colAcornLight       = color.NRGBA{R: 223, G: 151, B: 66, A: 255}
	colAcornCap         = color.NRGBA{R: 101, G: 65, B: 40, A: 255}
	colBeetle           = color.NRGBA{R: 74, G: 122, B: 74, A: 255}
	colBeetleShell      = color.NRGBA{R: 112, G: 159, B: 87, A: 255}
	colBeetleDark       = color.NRGBA{R: 54, G: 86, B: 60, A: 255}
	colEye              = color.NRGBA{R: 255, G: 236, B: 153, A: 255}
	colInk              = color.NRGBA{R: 39, G: 48, B: 38, A: 255}
	colFur              = color.NRGBA{R: 169, G: 102, B: 58, A: 255}
	colFurLight         = color.NRGBA{R: 224, G: 167, B: 109, A: 255}
	colFurDark          = color.NRGBA{R: 113, G: 68, B: 48, A: 255}
	colVest             = color.NRGBA{R: 53, G: 112, B: 103, A: 255}
	colVestLight        = color.NRGBA{R: 91, G: 157, B: 124, A: 255}
	colBoot             = color.NRGBA{R: 75, G: 56, B: 45, A: 255}
	colTail             = color.NRGBA{R: 117, G: 71, B: 50, A: 255}
	colTailLight        = color.NRGBA{R: 191, G: 123, B: 67, A: 255}
	colBark             = color.NRGBA{R: 118, G: 76, B: 48, A: 255}
	colBarkDark         = color.NRGBA{R: 76, G: 49, B: 40, A: 255}
	colDoorway          = color.NRGBA{R: 53, G: 83, B: 61, A: 255}
	colWindow           = color.NRGBA{R: 240, G: 207, B: 113, A: 255}
)
