package bombsweeper

import (
	"math/rand"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
)

const (
	tileSize = 40
)

// Size stores the game dimensions
type Size struct {
	Rows  int
	Cols  int
	Mines int
}

// Premade difficulty levels
var (
	SizeBeginner = Size{
		Rows:  9,
		Cols:  9,
		Mines: 10,
	}

	SizeIntermediate = Size{
		Rows:  16,
		Cols:  16,
		Mines: 40,
	}

	SizeExpert = Size{
		Rows:  16,
		Cols:  30,
		Mines: 99,
	}
)

type Game struct {
	Board       *fyne.Container
	Tiles       [][]*Tile
	Smiley      *SmileyMan
	MineCounter *MineCounter
	TimeCounter *TimeCounter

	OpenCount int
	WinCount  int
	IsRunning bool
	Size      Size

	Win fyne.Window
}

// NewGame creates a new game
func NewGame() *Game {
	return &Game{}
}

// SeedGame creates a new game damnit
func (g *Game) SeedGame(size Size) {
	plan := make([][]int, 0)

	// generate a blank plan
	for itrRow := 0; itrRow < size.Rows; itrRow++ {
		row := make([]int, 0)
		for itrCol := 0; itrCol < size.Cols; itrCol++ {
			row = append(row, 0)
		}
		plan = append(plan, row)
	}

	// generate mines with surrounding info
	plan = generatePlan(plan, size.Mines)

	// generate game tiles based on plan
	tiles := make([][]*Tile, 0)
	for itrRow := 0; itrRow < size.Rows; itrRow++ {
		row := make([]*Tile, 0)
		for itrCol := 0; itrCol < size.Cols; itrCol++ {
			var tile *Tile
			switch plan[itrRow][itrCol] {
			case -1:
				tile = NewTile(TileTypeMine, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 0:
				tile = NewTile(TileType0, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 1:
				tile = NewTile(TileType1, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 2:
				tile = NewTile(TileType2, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 3:
				tile = NewTile(TileType3, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 4:
				tile = NewTile(TileType4, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 5:
				tile = NewTile(TileType5, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 6:
				tile = NewTile(TileType6, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 7:
				tile = NewTile(TileType7, itrRow, itrCol, g.openTile, g.markTile)
				break
			case 8:
				tile = NewTile(TileType8, itrRow, itrCol, g.openTile, g.markTile)
				break
			default:
			}

			row = append(row, tile)
		}
		tiles = append(tiles, row)
	}

	board := NewBoard(tiles)
	sm := NewSmileyMan(GameStateOngoing, g.resetHandler)
	mc := NewMineCounter(size.Mines)
	tc := NewTimeCounter(0)

	g.Board = board
	g.Tiles = tiles
	g.Smiley = sm
	g.MineCounter = mc
	g.TimeCounter = tc
	g.OpenCount = 0
	g.IsRunning = true
	g.Size = size
	g.WinCount = (size.Rows * size.Cols) - size.Mines
}

func generatePlan(plan [][]int, mineCount int) [][]int {
	rand.Seed(time.Now().UnixNano())
	maxRow := len(plan) - 1
	maxCol := len(plan[0]) - 1
	itr := 0
	for {
		if itr == mineCount {
			break
		}

		r := rand.Intn(maxRow + 1)
		c := rand.Intn(maxCol + 1)
		if isMine(plan, r, c) {
			continue
		}
		plan[r][c] = -1
		if isInBounds(r-1, c-1, 0, 0, maxRow, maxCol) && !isMine(plan, r-1, c-1) {
			plan[r-1][c-1]++
		}
		if isInBounds(r-1, c, 0, 0, maxRow, maxCol) && !isMine(plan, r-1, c) {
			plan[r-1][c]++
		}
		if isInBounds(r-1, c+1, 0, 0, maxRow, maxCol) && !isMine(plan, r-1, c+1) {
			plan[r-1][c+1]++
		}
		if isInBounds(r, c-1, 0, 0, maxRow, maxCol) && !isMine(plan, r, c-1) {
			plan[r][c-1]++
		}
		if isInBounds(r, c+1, 0, 0, maxRow, maxCol) && !isMine(plan, r, c+1) {
			plan[r][c+1]++
		}
		if isInBounds(r+1, c-1, 0, 0, maxRow, maxCol) && !isMine(plan, r+1, c-1) {
			plan[r+1][c-1]++
		}
		if isInBounds(r+1, c, 0, 0, maxRow, maxCol) && !isMine(plan, r+1, c) {
			plan[r+1][c]++
		}
		if isInBounds(r+1, c+1, 0, 0, maxRow, maxCol) && !isMine(plan, r+1, c+1) {
			plan[r+1][c+1]++
		}

		itr++
	}

	return plan
}

// Render the game
func (g *Game) Render() {
	mc := container.NewCenter(g.MineCounter.Container)
	tc := container.NewCenter(g.TimeCounter.Container)
	topBar := container.NewBorder(nil, nil, mc, tc, g.Smiley)
	t := container.NewVBox(topBar, g.Board)
	content := container.NewCenter(t)
	minSize := content.MinSize()
	currentSize := g.Win.Canvas().Size()

	g.Win.SetContent(content)
	g.Win.Resize(fyne.NewSize(
		max(minSize.Width, currentSize.Width),
		max(minSize.Height, currentSize.Height),
	))
}

func (g *Game) resetHandler() {
	g.SeedGame(g.Size)
	g.Render()
}

func isInBounds(row, col, minRow, minCol, maxRow, maxCol int) bool {
	return row >= minRow && row <= maxRow && col >= minCol && col <= maxCol
}

func isMine(plan [][]int, row, col int) bool {
	return plan[row][col] == -1
}
