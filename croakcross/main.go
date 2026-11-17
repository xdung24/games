// CroakCross — a nostalgic single-player Frogger clone built with Fyne.
//
// Hop the frog across five lanes of traffic, over the median, then ride logs
// and turtles across the river to one of the five home bays at the top. Fill
// all five bays to clear the level; each level runs faster and more turtles
// dive. Watch the clock, catch the occasional fly for a bonus, and don't land
// on a crocodile.
//
// Controls:
//
//	← ↑ → ↓ / W A S D        hop
//	Tap (towards the frog) / swipe   hop on touch screens
//	Space / Enter / tap      start · pause/resume · play again
//	P                        pause / resume
//	R                        restart
//	M                        mute / unmute the sound
//
// Play also pauses automatically whenever the app leaves the foreground.
package croakcross

import (
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const targetFPS = 60

// prefKeyMuted is the preferences key remembering whether sound is muted.
const prefKeyMuted = "sound.muted"

func main() {
	a := app.New()
	a.Settings().SetTheme(newGameTheme())
	w := a.NewWindow("CroakCross")
	w.SetPadded(false)

	game := NewGame()
	sound := newSoundPlayer()
	sound.SetMuted(a.Preferences().Bool(prefKeyMuted)) // restore the saved choice
	game.onSound = sound.Play
	board := NewBoard(game)

	mutedIcon := widget.NewIcon(theme.VolumeMuteIcon())
	if !sound.Muted() {
		mutedIcon.Hide()
	}
	board.onToggleMute = func() {
		muted := !sound.Muted()
		sound.SetMuted(muted)
		a.Preferences().SetBool(prefKeyMuted, muted)
		if muted {
			mutedIcon.Show()
		} else {
			mutedIcon.Hide()
		}
	}

	overlay := container.NewVBox(layout.NewSpacer(),
		container.NewHBox(layout.NewSpacer(), mutedIcon))
	w.SetContent(container.NewStack(board, overlay))
	w.Resize(fyne.NewSize(572, 660))

	// Give the board keyboard focus so the arrow keys reach it.
	w.Canvas().Focus(board)

	a.Lifecycle().SetOnExitedForeground(func() {
		game.Pause()
		board.Refresh()
	})

	startLoop(game, board)
	w.ShowAndRun()
}

// startLoop drives the simulation on a fixed-rate ticker. All state changes and
// the canvas refresh are marshalled onto the UI goroutine via fyne.Do, so the
// model is only ever touched from one thread and needs no locking.
func startLoop(game *Game, board *Board) {
	go func() {
		ticker := time.NewTicker(time.Second / targetFPS)
		defer ticker.Stop()
		last := time.Now()
		for now := range ticker.C {
			dt := float32(now.Sub(last).Seconds())
			last = now
			if dt > 0.05 { // clamp after a stall so nothing teleports
				dt = 0.05
			}
			fyne.Do(func() {
				// Do the least work the frame needs: nothing when idle, a cheap
				// reposition while things move, and a full redraw only when
				// something structural (score, homes, banner) changed.
				if !board.Visible() {
					return
				}
				switch game.Tick(dt) {
				case RenderMove:
					board.MoveDynamic()
				case RenderFull:
					board.Refresh()
				}
			})
		}
	}()
}
