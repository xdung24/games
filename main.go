package main

import (
	_ "embed"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	asteroidshooter "github.com/xdung24/games/asteroidshooter"
	brickbreaker "github.com/xdung24/games/brickbreaker"
	flappychick "github.com/xdung24/games/flappychick"
	flappygophy "github.com/xdung24/games/flappygophy"
)

//go:embed Icon.png
var homeIconPNG []byte
var homeIcon = fyne.NewStaticResource("Home-Icon.png", homeIconPNG)

//go:embed asteroidshooter/Icon.png
var asteroidShooterIconPNG []byte
var asteroidShooterIcon = fyne.NewStaticResource("asteroid-shooter-Icon.png", asteroidShooterIconPNG)

//go:embed brickbreaker/Icon.png
var brickBreakerIconPNG []byte
var brickBreakerIcon = fyne.NewStaticResource("brick-breaker-Icon.png", brickBreakerIconPNG)

//go:embed flappygophy/Icon.png
var flappyGophyIconPNG []byte
var flappyGophyIcon = fyne.NewStaticResource("flappy-gophy-Icon.png", flappyGophyIconPNG)

//go:embed flappychick/Icon.svg
var flappyChickIconPNG []byte
var flappyChickIcon = fyne.NewStaticResource("flappy-chick-Icon.svg", flappyChickIconPNG)

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
	var asteroidshooterScreen *asteroidshooter.Screen
	var flappyScreen *flappychick.Screen
	var gophyScreen *flappygophy.Screen
	showMenu := func() {
		a.Settings().SetTheme(brickbreaker.NewTheme())
		w.SetTitle("Play game")
		menu.Show(w)
	}

	menu = newGameMenu([]gameEntry{
		{
			name:        "Asteroid Shooter",
			description: "Destroy the stars and reach the highest score.",
			icon:        asteroidShooterIcon,
			play: func() {
				a.Settings().SetTheme(asteroidshooter.NewTheme())
				w.SetTitle("Asteroid Shooter")
				if asteroidshooterScreen == nil {
					asteroidshooterScreen = asteroidshooter.NewScreen(w, a.Preferences(), showMenu)
				}
				asteroidshooterScreen.Show()
			},
		},
		{
			name:        "Brick Breaker",
			description: "Keep the ball in play and break the ocean-blue blocks.",
			icon:        brickBreakerIcon,
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
			name:        "Flappy Chick",
			description: "Tap to flap, slip through the pipes, and beat your best score.",
			icon:        flappyChickIcon,
			play: func() {
				a.Settings().SetTheme(brickbreaker.NewTheme())
				w.SetTitle("Flappy Chick")
				if flappyScreen == nil {
					flappyScreen = flappychick.NewScreen(w, a.Preferences(), showMenu)
				}
				flappyScreen.Show()
			},
		},
		{
			name:        "Flappy Gophy",
			description: "Guide a hyperactive Gopher through the pipes.",
			icon:        flappyGophyIcon,
			play: func() {
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
		if asteroidshooterScreen != nil {
			asteroidshooterScreen.Pause()
		}
		if brickScreen != nil {
			brickScreen.Pause()
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
