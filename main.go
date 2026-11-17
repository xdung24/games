package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	brickbreaker "github.com/xdung24/game/brickbreaker"
	starbreaker "github.com/xdung24/game/starbreaker"
)

//go:embed Icon.png
var homeIconPNG []byte

//go:embed brickbreaker/Icon.png
var brickBreakerIconPNG []byte

//go:embed starbreaker/Icon.png
var starBreakerIconPNG []byte

var homeIcon = fyne.NewStaticResource("Icon.png", homeIconPNG)

type gameEntry struct {
	name        string
	description string
	icon        fyne.Resource
	play        func()
}

func main() {
	a := app.New()
	a.Settings().SetTheme(brickbreaker.NewTheme())
	w := a.NewWindow("game-window")
	w.SetPadded(false)

	var menu *gameMenu
	var brickScreen *brickbreaker.Screen
	var starScreen *starbreaker.Screen
	showMenu := func() {
		a.Settings().SetTheme(brickbreaker.NewTheme())
		w.SetTitle("Play game")
		menu.Show(w)
	}

	menu = newGameMenu([]gameEntry{
		{
			name:        "Brick Breaker",
			description: "Keep the ball in play and break the ocean-blue blocks.",
			icon:        fyne.NewStaticResource("brick-breaker-Icon.png", brickBreakerIconPNG),
			play: func() {
				a.Settings().SetTheme(brickbreaker.NewTheme())
				w.SetTitle("Brick Breaker")
				if brickScreen == nil {
					brickScreen = brickbreaker.NewScreen(w, a.Preferences(), showMenu)
				}
				brickScreen.Show()
			},
		},
		{
			name:        "Star Breaker",
			description: "Destroy the stars and reach the highest score.",
			icon:        fyne.NewStaticResource("star-breaker-Icon.png", starBreakerIconPNG),
			play: func() {
				a.Settings().SetTheme(starbreaker.NewTheme())
				w.SetTitle("Star Breaker")
				if starScreen == nil {
					starScreen = starbreaker.NewScreen(w, a.Preferences(), showMenu)
				}
				starScreen.Show()
			},
		}})
	w.Resize(fyne.NewSize(640, 600))
	showMenu()

	a.Lifecycle().SetOnExitedForeground(func() {
		if brickScreen != nil {
			brickScreen.Pause()
		}
		if starScreen != nil {
			starScreen.Pause()
		}
	})
	w.ShowAndRun()
}
