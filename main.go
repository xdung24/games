package main

import (
	_ "embed"
	"runtime"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"

	asteroidshooter "github.com/xdung24/games/asteroidshooter"
	bombsweeper "github.com/xdung24/games/bombsweeper"
	brickbreaker "github.com/xdung24/games/brickbreaker"
	croakcross "github.com/xdung24/games/croakcross"
	flappychick "github.com/xdung24/games/flappychick"
	flappygophy "github.com/xdung24/games/flappygophy"
	"github.com/xdung24/games/moria"
)

//go:embed moria/Icon.svg
var moriaIconSVG []byte
var moriaIcon = fyne.NewStaticResource("moria-Icon.svg", moriaIconSVG)

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

//go:embed bombsweeper/Icon.png
var bombsweeperIconPNG []byte
var bombsweeperIcon = fyne.NewStaticResource("bombsweeper-Icon.png", bombsweeperIconPNG)

//go:embed croakcross/Icon.png
var croakcrossIconPNG []byte
var croakcrossIcon = fyne.NewStaticResource("croakcross-Icon.png", croakcrossIconPNG)

type gameEntry struct {
	name        string
	description string
	icon        fyne.Resource
	isDesktop   bool
	isMobile    bool
	isWeb       bool
	play        func()
}

func newGameEntries(a fyne.App, w fyne.Window, showMenu func(), targetOS string) ([]gameEntry, func()) {
	var brickScreen *brickbreaker.Screen
	var asteroidshooterScreen *asteroidshooter.Screen
	var croakcrossScreen *croakcross.Screen
	var flappyScreen *flappychick.Screen
	var gophyScreen *flappygophy.Screen
	var moriaScreen *moria.Screen

	entries := []gameEntry{
		{
			name:        "Asteroid Shooter",
			description: "Destroy the asteroids and reach the highest score.",
			icon:        asteroidShooterIcon,
			isDesktop:   true,
			isMobile:    true,
			isWeb:       true,
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
			isDesktop:   true,
			isMobile:    true,
			isWeb:       true,
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
			name:        "Bombsweeper",
			description: "Clear the board without uncovering bombs.",
			icon:        bombsweeperIcon,
			isDesktop:   true,
			isMobile:    false,
			isWeb:       false,
			play: func() {
				a.Settings().SetTheme(bombsweeper.NewTheme())
				w.SetTitle("Bombsweeper")
				game := bombsweeper.NewGame()
				game.Win = w
				game.SeedGame(bombsweeper.SizeBeginner)
				w.SetMainMenu(bombsweeper.NewMenuBar(game, func() {
					game.IsRunning = false
					showMenu()
				}))
				game.Render()
			},
		},
		{
			name:        "CroakCross",
			description: "Hop across traffic and ride the river platforms to safety.",
			icon:        croakcrossIcon,
			isDesktop:   true,
			isMobile:    true,
			isWeb:       true,
			play: func() {
				a.Settings().SetTheme(croakcross.NewTheme())
				w.SetTitle("CroakCross")
				if croakcrossScreen == nil {
					croakcrossScreen = croakcross.NewScreen(w, a.Preferences(), showMenu)
				}
				croakcrossScreen.Show()
			},
		},
		{
			name:        "Flappy Chick",
			description: "Tap to flap, slip through the pipes, and beat your best score.",
			icon:        flappyChickIcon,
			isDesktop:   true,
			isMobile:    true,
			isWeb:       true,
			play: func() {
				a.Settings().SetTheme(flappychick.NewTheme())
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
			isDesktop:   true,
			isMobile:    true,
			isWeb:       true,
			play: func() {
				a.Settings().SetTheme(brickbreaker.NewTheme())
				w.SetTitle("Flappy Gophy")
				if gophyScreen == nil {
					gophyScreen = flappygophy.NewScreen(w, a.Preferences(), showMenu)
				}
				gophyScreen.Show()
			},
		},
		{
			name:        "Moria",
			description: "Explore the crystal caverns, gather gems, and find the ancient gate.",
			icon:        moriaIcon,
			isDesktop:   true,
			isMobile:    true,
			isWeb:       true,
			play: func() {
				a.Settings().SetTheme(moria.NewTheme())
				w.SetTitle("Moria")
				if moriaScreen == nil {
					moriaScreen = moria.NewScreen(w, a.Preferences(), showMenu)
				}
				moriaScreen.Show()
			},
		},
	}

	pauseGames := func() {
		if asteroidshooterScreen != nil {
			asteroidshooterScreen.Pause()
		}
		if brickScreen != nil {
			brickScreen.Pause()
		}
		if croakcrossScreen != nil {
			croakcrossScreen.Pause()
		}
		if flappyScreen != nil {
			flappyScreen.Pause()
		}
		if gophyScreen != nil {
			gophyScreen.Pause()
		}
		if moriaScreen != nil {
			moriaScreen.Pause()
		}
	}

	return gamesForPlatform(entries, targetOS), pauseGames
}

func gamesForPlatform(entries []gameEntry, targetOS string) []gameEntry {
	available := make([]gameEntry, 0, len(entries))
	for _, entry := range entries {
		var supported bool
		switch targetOS {
		case "js":
			supported = entry.isWeb
		case "android", "ios":
			supported = entry.isMobile
		default:
			supported = entry.isDesktop
		}
		if supported {
			available = append(available, entry)
		}
	}
	return available
}

func main() {
	a := app.NewWithID("com.dungle.game")
	a.Settings().SetTheme(brickbreaker.NewTheme())
	w := a.NewWindow("game-window")
	w.SetPadded(false)

	var menu *gameMenu
	showMenu := func() {
		a.Settings().SetTheme(brickbreaker.NewTheme())
		w.SetTitle("Let's play game")
		w.SetMainMenu(nil)
		menu.Show(w)
	}

	entries, pauseGames := newGameEntries(a, w, showMenu, runtime.GOOS)
	menu = newGameMenu(entries)

	w.Resize(fyne.NewSize(640, 600))
	showMenu()

	a.Lifecycle().SetOnExitedForeground(func() {
		pauseGames()
	})
	w.ShowAndRun()
}
