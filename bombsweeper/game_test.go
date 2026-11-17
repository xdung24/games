package bombsweeper

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
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

func TestBaseResourceByType(t *testing.T) {
	tests := []struct {
		name   string
		typeID TileType
		byUser bool
		want   fyne.Resource
	}{
		{"clicked mine", TileTypeMine, true, resourceMineRedPng},
		{"revealed mine", TileTypeMine, false, resourceMinePng},
		{"zero", TileType0, false, resourceType0Png},
		{"one", TileType1, false, resourceType1Png},
		{"two", TileType2, false, resourceType2Png},
		{"three", TileType3, false, resourceType3Png},
		{"four", TileType4, false, resourceType4Png},
		{"five", TileType5, false, resourceType5Png},
		{"six", TileType6, false, resourceType6Png},
		{"seven", TileType7, false, resourceType7Png},
		{"eight", TileType8, false, resourceType8Png},
		{"empty", TileTypeEmpty, false, nil},
		{"unknown", TileType(99), false, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBaseResourceByType(tt.typeID, tt.byUser)
			if tt.want == nil {
				if got != nil {
					t.Fatalf("resource = %v, want nil", got)
				}
				return
			}
			if got == nil || got.Name() != tt.want.Name() {
				t.Fatalf("resource = %v, want %v", got, tt.want)
			}
		})
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

	center := app.Driver().AbsolutePositionForObject(tile).
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

func TestDifficultyKeepsMenuAndScrollableBoard(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(NewTheme())
	window := test.NewWindow(nil)
	defer window.Close()
	window.SetPadded(false)
	window.Resize(fyne.NewSize(640, 600))

	game := NewGame()
	game.Win = window
	game.SeedGame(SizeBeginner)
	menu := NewMenuBar(game, func() {})
	window.SetMainMenu(menu)
	game.Render()
	window.Show()
	initialSize := window.Canvas().Size()

	for _, tt := range []struct {
		index int
		size  Size
	}{
		{1, SizeIntermediate},
		{2, SizeExpert},
		{0, SizeBeginner},
	} {
		menu.Items[1].Items[tt.index].Action()
		if game.Size != tt.size || window.MainMenu() != menu {
			t.Fatalf("difficulty %v did not preserve the game menu", tt.size)
		}
		if got := window.Canvas().Size(); got != initialSize {
			t.Fatalf("difficulty %v resized canvas from %v to %v", tt.size, initialSize, got)
		}
		scroll, ok := window.Content().(*container.Scroll)
		if !ok {
			t.Fatal("game content should be scrollable")
		}
		if tt.size == SizeExpert {
			if scroll.Content.MinSize().Width <= scroll.Size().Width || scroll.Content.MinSize().Height <= scroll.Size().Height {
				t.Fatal("expert board should overflow the test window in both directions")
			}
			scroll.ScrollToOffset(fyne.NewPos(
				scroll.Content.MinSize().Width-scroll.Size().Width,
				scroll.Content.MinSize().Height-scroll.Size().Height,
			))
			if scroll.Offset.X <= 0 || scroll.Offset.Y <= 0 {
				t.Fatalf("expert board did not scroll in both directions: %v", scroll.Offset)
			}
			tile := game.Tiles[tt.size.Rows-1][tt.size.Cols-2]
			tile.Base = TileType1
			game.OpenCount = 1
			center := app.Driver().AbsolutePositionForObject(tile).
				Add(fyne.NewPos(tile.Size().Width/2, tile.Size().Height/2))
			test.TapCanvas(window.Canvas(), center)
			if !tile.IsOpen {
				t.Fatalf("scrolled tile should be clickable: center %v, offset %v, tile position %v, tile size %v, board pos %v size %v, scroll size %v, content size %v", center, scroll.Offset, tile.Position(), tile.Size(), game.Board.Position(), game.Board.Size(), scroll.Size(), scroll.Content.Size())
			}
		}
	}
}

func TestNewGameMenuRestartsCurrentDifficulty(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	window := test.NewWindow(nil)
	defer window.Close()

	game := NewGame()
	game.Win = window
	game.SeedGame(SizeIntermediate)
	game.Render()
	previousBoard, previousCounter, previousStop := game.Board, game.TimeCounter, game.timerStop
	game.OpenCount = 5
	game.IsRunning = false

	menu := NewMenuBar(game, func() {})
	if menu.Items[0].Label != "Game" || menu.Items[0].Items[0].Label != "New Game" {
		t.Fatal("expected Game > New Game menu item")
	}
	menu.Items[0].Items[0].Action()
	if game.Board == previousBoard || game.Size != SizeIntermediate || game.OpenCount != 0 || !game.IsRunning {
		t.Fatal("New Game should reset the board at the current difficulty")
	}
	select {
	case <-previousStop:
	default:
		t.Fatal("New Game should stop the previous timer")
	}
	game.startTimer(previousCounter, previousStop)
	if previousCounter.current != 0 || game.TimeCounter.current != 0 {
		t.Fatal("a stopped timer must not increment either counter")
	}
}

func TestThemeColors(t *testing.T) {
	for _, variant := range []fyne.ThemeVariant{theme.VariantLight, theme.VariantDark} {
		for _, name := range []fyne.ThemeColorName{theme.ColorNameBackground, theme.ColorNameMenuBackground} {
			got := color.NRGBAModel.Convert(NewTheme().Color(name, variant)).(color.NRGBA)
			if got.A != 255 || got.R <= got.G || got.G <= got.B {
				t.Errorf("%s: background = %v, want opaque earthy brown", name, got)
			}
		}
		if got := color.NRGBAModel.Convert(NewTheme().Color(theme.ColorNameForeground, variant)).(color.NRGBA); got != (color.NRGBA{A: 255}) {
			t.Errorf("foreground = %v, want black", got)
		}
	}
}

func TestBoardBorder(t *testing.T) {
	app := test.NewApp()
	defer app.Quit()
	app.Settings().SetTheme(NewTheme())
	tile := NewTile(TileType0, 0, 0, nil, nil)
	board := NewBoard([][]*Tile{{tile}})
	window := test.NewWindow(board)
	defer window.Close()
	window.SetPadded(false)
	window.Resize(board.MinSize())
	window.Show()

	img := window.Canvas().Capture()
	borderColor := color.NRGBA{R: 112, G: 84, B: 59, A: 255}
	width, height := img.Bounds().Dx(), img.Bounds().Dy()
	for _, point := range [][2]int{{0, 0}, {width - 1, 0}, {0, height - 1}, {width - 1, height - 1}, {width / 2, 0}, {width / 2, height - 1}, {0, height / 2}, {width - 1, height / 2}} {
		if got := color.NRGBAModel.Convert(img.At(point[0], point[1])).(color.NRGBA); got != borderColor {
			t.Errorf("border pixel at %v = %v, want %v", point, got, borderColor)
		}
	}
	if tile.Size().Width < tileSize || tile.Size().Height < tileSize {
		t.Errorf("tile size = %v, want at least %d x %d", tile.Size(), tileSize, tileSize)
	}
}
