package bombsweeper

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

const boardBorderWidth = 3

// NewBoard creates a new board of dimensions rows and cols
func NewBoard(tiles [][]*Tile) *fyne.Container {
	border := func(width, height float32) *canvas.Rectangle {
		edge := canvas.NewRectangle(color.NRGBA{R: 112, G: 84, B: 59, A: 255})
		edge.SetMinSize(fyne.NewSize(width, height))
		return edge
	}
	grid := container.NewGridWithRows(len(tiles), flattenTiles(tiles)...)
	return container.NewBorder(
		border(0, boardBorderWidth), border(0, boardBorderWidth),
		border(boardBorderWidth, 0), border(boardBorderWidth, 0), grid,
	)
}

func flattenTiles(tiles [][]*Tile) []fyne.CanvasObject {
	res := make([]fyne.CanvasObject, 0)

	for _, row := range tiles {
		for _, item := range row {
			res = append(res, item)
		}
	}

	return res
}
