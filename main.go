package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	brickbreaker "github.com/xdung24/games/brickbreaker"
	flappybird "github.com/xdung24/games/flappybird"
	flappygophy "github.com/xdung24/games/flappygophy"
	browserinput "github.com/xdung24/games/internal/browserinput"
	starbreaker "github.com/xdung24/games/starbreaker"
)

//go:embed Icon.png
var homeIconPNG []byte

//go:embed brickbreaker/Icon.png
var brickBreakerIconPNG []byte

//go:embed starbreaker/Icon.png
var starBreakerIconPNG []byte

//go:embed flappygophy/Icon.png
var flappyGophyIconPNG []byte

var homeIcon = fyne.NewStaticResource("Icon.png", homeIconPNG)

type gameEntry struct {
	name        string
	description string
	icon        fyne.Resource
	play        func()
}

func main() {
	a := app.NewWithID("com.dungle.game")
	a.Settings().SetTheme(brickbreaker.NewTheme())
	w := a.NewWindow("game-window")
	w.SetPadded(false)

	var menu *gameMenu
	var brickScreen *brickbreaker.Screen
	var starScreen *starbreaker.Screen
	var flappyScreen *flappybird.Screen
	var gophyScreen *flappygophy.Screen
	showMenu := func() {
		browserinput.Show("")
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
				browserinput.Show("")
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
				browserinput.Show("starbreaker")
				a.Settings().SetTheme(starbreaker.NewTheme())
				w.SetTitle("Star Breaker")
				if starScreen == nil {
					starScreen = starbreaker.NewScreen(w, a.Preferences(), showMenu)
				}
				starScreen.Show()
			},
		},
		{
			name:        "Flappy Bird",
			description: "Tap to flap, slip through the pipes, and beat your best score.",
			icon:        flappybird.Icon,
			play: func() {
				browserinput.Show("")
				a.Settings().SetTheme(brickbreaker.NewTheme())
				w.SetTitle("Flappy Bird")
				if flappyScreen == nil {
					flappyScreen = flappybird.NewScreen(w, a.Preferences(), showMenu)
				}
				flappyScreen.Show()
			},
		},
		{
			name:        "Flappy Gophy",
			description: "Guide a hyperactive Gopher through the pipes.",
			icon:        fyne.NewStaticResource("flappy-gophy-Icon.png", flappyGophyIconPNG),
			play: func() {
				browserinput.Show("")
				a.Settings().SetTheme(brickbreaker.NewTheme())
				w.SetTitle("Flappy Gophy")
				if gophyScreen == nil {
					gophyScreen = flappygophy.NewScreen(w, a.Preferences(), showMenu)
				}
				gophyScreen.Show()
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
		if flappyScreen != nil {
			flappyScreen.Pause()
		}
		if gophyScreen != nil {
			gophyScreen.Pause()
		}
	})
	w.ShowAndRun()
}
