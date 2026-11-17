package bombsweeper

import (
	"fyne.io/fyne/v2"
)

func NewMenuBar(g *Game, onBack func()) *fyne.MainMenu {
	miNewGame := fyne.NewMenuItem("New Game", g.resetHandler)
	miHome := fyne.NewMenuItem("Home", onBack)
	miSizeBeginner := fyne.NewMenuItem("Beginner", func() {
		g.SeedGame(SizeBeginner)
		g.Render()
	})
	miSizeIntermediate := fyne.NewMenuItem("Intermediate", func() {
		g.SeedGame(SizeIntermediate)
		g.Render()
	})
	miSizeExpert := fyne.NewMenuItem("Expert", func() {
		g.SeedGame(SizeExpert)
		g.Render()
	})

	mGame := fyne.NewMenu("Game", miNewGame, miHome)
	mDifficulty := fyne.NewMenu("Difficulty", miSizeBeginner, miSizeIntermediate, miSizeExpert)
	mm := fyne.NewMainMenu(mGame, mDifficulty)
	return mm
}
