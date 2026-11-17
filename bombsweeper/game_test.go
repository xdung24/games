package bombsweeper

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
)

func TestOpenedCellColors(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	cells := []struct {
		name string
		base TileType
		ink  color.NRGBA
	}{
		{"one", TileType1, color.NRGBA{B: 255, A: 255}},
		{"two", TileType2, color.NRGBA{G: 128, A: 255}},
		{"three", TileType3, color.NRGBA{R: 255, A: 255}},
		{"four", TileType4, color.NRGBA{B: 128, A: 255}},
		{"five", TileType5, color.NRGBA{R: 128, A: 255}},
		{"six", TileType6, color.NRGBA{G: 128, B: 128, A: 255}},
		{"seven", TileType7, color.NRGBA{A: 255}},
		{"eight", TileType8, color.NRGBA{R: 128, G: 128, B: 128, A: 255}},
	}
	for _, cellTheme := range []fyne.Theme{theme.LightTheme(), theme.DarkTheme()} {
		app.Settings().SetTheme(cellTheme)
		for _, cell := range cells {
			tile := NewTile(cell.base, 0, 0, nil, nil)
			window := test.NewWindow(tile)
			window.SetPadded(false)
			window.Resize(fyne.NewSize(tileSize, tileSize))
			tile.Open(true)
			window.Show()
			img := window.Canvas().Capture()
			gray, ink := 0, 0
			for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
				for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
					c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
					if c.R == 198 && c.G == 198 && c.B == 198 {
						gray++
					}
					if c == cell.ink {
						ink++
					}
				}
			}
			window.Close()
			if gray < 500 || ink < 20 {
				t.Errorf("%s: opened cell has %d gray pixels and %d number-colored pixels, want gray background and readable number", cell.name, gray, ink)
			}
		}
	}
}

func TestCanvasClickOpensTile(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	window := test.NewWindow(nil)
	defer window.Close()
	window.Resize(fyne.NewSize(640, 600))

	game := NewGame()
	game.Win = window
	game.SeedGame(SizeBeginner)
	game.Render()
	window.Show()

	var tile *Tile
	for _, row := range game.Tiles {
		for _, candidate := range row {
			if candidate.Base != TileTypeMine {
				tile = candidate
				break
			}
		}
		if tile != nil {
			break
		}
	}

	if tile == nil {
		t.Fatal("expected a safe tile")
	}
	if tile.Size().Width < tileSize || tile.Size().Height < tileSize {
		t.Fatalf("tile size = %v, want at least %d x %d", tile.Size(), tileSize, tileSize)
	}
	if size := window.Canvas().Size(); size.Width < 640 || size.Height < 600 {
		t.Fatalf("game render shrank the desktop window to %v", size)
	}

	test.TapSecondary(tile)
	if !tile.IsFlagged {
		t.Fatal("right-clicking a closed tile should flag it")
	}
	test.TapSecondary(tile)

	gameContent := window.Canvas().Content().(*fyne.Container).Objects[0]
	center := gameContent.Position().Add(game.Board.Position()).Add(tile.Position()).
		Add(fyne.NewPos(tile.Size().Width/2, tile.Size().Height/2))
	test.TapCanvas(window.Canvas(), center)
	if !tile.IsOpen {
		t.Fatalf("clicking a safe tile should open it (tile position %v, size %v, center %v)", tile.Position(), tile.Size(), center)
	}
	game.IsRunning = false
}

func TestSmileyTapResetsGame(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	window := test.NewWindow(nil)
	defer window.Close()

	game := NewGame()
	game.Win = window
	game.SeedGame(SizeBeginner)
	game.Render()
	previousBoard := game.Board

	test.Tap(game.Smiley)
	if game.Board == previousBoard {
		t.Fatal("tapping the smiley should start a new board")
	}
}

func TestThemeBackgroundIsOpaqueGray(t *testing.T) {
	r, g, b, a := NewTheme().Color(theme.ColorNameBackground, theme.VariantLight).RGBA()
	if a != 0xffff || r != 0xc0c0 || g != 0xc0c0 || b != 0xc0c0 {
		t.Fatalf("background color = %#04x%#04x%#04x%#04x, want opaque gray", r, g, b, a)
	}
}
