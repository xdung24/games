package bombsweeper

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/widget"
)

// TileTypes
const (
	TileTypeMine = iota
	TileTypeEmpty
	TileType0
	TileType1
	TileType2
	TileType3
	TileType4
	TileType5
	TileType6
	TileType7
	TileType8
)

// TileType is the type of the tile that modifies it's look
type TileType int

// Tile stores the data and state of a tile on the game board
type Tile struct {
	widget.Icon

	Base      TileType
	IsOpen    bool
	IsFlagged bool
	Row       int
	Col       int

	OpenHandler func(int, int)
	MarkHandler func(int, int)
}

// NewTile creates a new tile of the given type
func NewTile(tileType TileType, row, col int, openHandler func(int, int), markHandler func(int, int)) *Tile {
	t := &Tile{Base: tileType, IsOpen: false, Row: row, Col: col, OpenHandler: openHandler, MarkHandler: markHandler}
	t.ExtendBaseWidget(t)
	t.SetResource(resourceClosedPng)
	return t
}

func (t *Tile) MinSize() fyne.Size {
	return fyne.NewSize(tileSize, tileSize)
}

// Open opens a tile
func (t *Tile) Open(byUser bool) {
	if !t.IsOpen {
		t.open(byUser)
	}
}

func (t *Tile) Tapped(*fyne.PointEvent) {
	if !t.IsOpen && !t.IsFlagged {
		t.OpenHandler(t.Row, t.Col)
	}
}

func (t *Tile) TappedSecondary(*fyne.PointEvent) {
	if !t.IsOpen {
		t.MarkHandler(t.Row, t.Col)
	}
}

// Flag falgs a tile
func (t *Tile) Flag() {
	t.SetResource(resourceFlagPng)
	t.IsFlagged = true
}

// Unflag unflags a tile
func (t *Tile) Unflag() {
	t.SetResource(resourceClosedPng)
	t.IsFlagged = false
}

func (t *Tile) open(byUser bool) {
	t.SetResource(getBaseResourceByType(t.Base, byUser))
	t.IsOpen = true
}

var numberResources = [...]fyne.Resource{
	resourceType0Png,
	resourceType1Png,
	resourceType2Png,
	resourceType3Png,
	resourceType4Png,
	resourceType5Png,
	resourceType6Png,
	resourceType7Png,
	resourceType8Png,
}

func getBaseResourceByType(tileType TileType, byUser bool) fyne.Resource {
	if tileType == TileTypeMine {
		if byUser {
			return resourceMineRedPng
		}
		return resourceMinePng
	}

	if tileType < TileType0 || tileType > TileType8 {
		return nil
	}

	return numberResources[tileType-TileType0]
}
