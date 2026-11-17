# GitHub Copilot instructions

## Project overview

This repository is a Go arcade-game launcher built with Fyne. It runs as a
desktop application and as WebAssembly in a browser. Use the Go version and
dependency versions declared in [go.mod](../go.mod).

## Repository structure

- [main.go](../main.go): application/window setup, embedded icons, game entries,
  lazy creation of game screens, theme switching, and foreground lifecycle.
- [menu.go](../menu.go): Home menu, pagination, keyboard selection, scrolling,
  and restoration of canvas key handlers.
- [card.go](../card.go): clickable game cards, accessibility, keyboard activation,
  and animated hover/focus highlights.
- [brickbreaker/](../brickbreaker) and [starbreaker/](../starbreaker): importable
  game packages, not standalone applications.
  - `game.go`: simulation, state transitions, physics, and sound-event callbacks.
  - `board.go`: Fyne widget, input handling, HUD, and canvas renderer.
  - `screen.go`: session ownership, navigation, focus, and simulation loop.
  - `sound.go`: synthesized effects and per-game mute handling.
  - `theme.go`: game-specific theme.
  - Star Breaker's `shaders.go`: desktop and OpenGL ES background shaders.
- [internal/audio/](../internal/audio): the shared Oto audio context.
- [internal/navigation/](../internal/navigation): shared double-Esc handling.
- [index.html](../index.html): full-window iframe loading `wasm/index.html`.
- [Games.toml](../Games.toml): application metadata; the launcher currently
  registers games explicitly in `main.go`, not by reading this file.

## Architecture and lifecycle

- Keep simulation independent of Fyne widgets and audio-device initialization.
  Models emit sound events through callbacks so simulation tests remain silent.
- Follow the existing screen API:
  `NewScreen(fyne.Window, fyne.Preferences, func()) *Screen`, `Show()`, `Pause()`,
  and `Hide()`. Export `NewTheme()` for launcher theme selection.
- Create each screen lazily once and reuse it. Returning Home pauses and hides
  the session; reopening it preserves progress and restores board focus.
- Use the screen's return callback rather than importing the root launcher from
  a game package. Restore the Home title, theme, and menu through the launcher.
- Pause initialized games when the application leaves the foreground. Release
  held controls and reset pending navigation when a board loses focus.
- Simulation loops target 60 FPS and clamp elapsed time to 0.05 seconds.
  Schedule model updates and canvas changes from background tickers with
  `fyne.Do`; do not mutate game or widget state directly from ticker goroutines.
- Skip simulation work while a board is hidden. Preserve Brick Breaker's
  `RenderNone`/`RenderMove`/`RenderFull` optimization rather than forcing a full
  redraw for every movement.
- Stop animations when their renderer is destroyed or their owning screen
  hides, as appropriate; avoid registering duplicate animations on reopening.

## Input and visual behavior

- Preserve the menu's 10-games-per-page behavior. Up/Down selects a game within
  the current page; Left/Right changes pages without wrapping past boundaries.
- Enter, keypad Enter, and Space launch the focused game card. Keep whole-card
  pointer activation, Tab navigation, and accessibility labels working.
- Keep keyboard selection visible by scrolling it into view, and restore the
  selection when returning Home. Restore prior canvas key handlers when leaving
  the menu so menu navigation cannot intercept gameplay input.
- Display the root `Icon.png` above the Home subtitle and each game's `Icon.png`
  to the left of its card's title and description. Embed launcher icons with
  `go:embed` and preserve their aspect ratios.
- Use the small, low-importance back-arrow button beside SCORE in each game.
  Keep it clickable above pause/game-over overlays and clear of other HUD text.
  Create it with `widget.NewButtonWithIcon("", theme.NavigateBackIcon(), ...)`
  and `widget.LowImportance`; do not add a separate Home toolbar or text label.
  Size it using `MinSize()` and position SCORE after the button plus theme
  padding so they do not overlap at narrow window sizes.
  A single click must call the board's `onBack` callback without also activating
  gameplay. Wire this callback in `NewScreen` to call `Screen.Hide()` before
  the launcher's Home callback, preserving progress and releasing held controls.
  Keep the button available in ready/attract, playing, paused, and game-over
  states. Test its placement and navigation above overlays.
- Reuse `navigation.EscapeHandler`: the first Esc pauses active play, and the
  second separate Esc press returns Home. Key repeat must not count as another
  press; other key input, pointer activation, and focus loss reset the sequence.
- Use theme colors for launcher highlights, preserve smooth fade reversal,
  and honor `Settings().ShowAnimations()`.

## Audio and browser compatibility

- Obtain the audio device through `audio.Context()`; Oto does not support a
  separate context for every game. Match its 44100 Hz, mono, signed-16-bit format.
- Persist mute settings through `fyne.Preferences`; do not rename existing
  preference keys without accounting for saved values.
- Keep games playable without an audio device and log initialization failures.
  Browser audio may require user interaction.
- Preserve desktop and browser support for input and rendering. Test browser
  controls rather than assuming letter-key shortcuts are delivered by the
  Fyne web driver. Retain both desktop and OpenGL ES shader variants.
- Edit the root web entry page, not generated files in `wasm`. Preserve the
  iframe's relative URL and autoplay permission.

## Adding a game

1. Add an importable package with the model/board/screen structure above,
   an `Icon.png`, a README describing controls, and focused tests.
2. Embed its icon and add a `gameEntry` in `main.go`.
3. Lazily create and retain its screen, select its theme on launch, and pass the
   shared Home callback to `NewScreen`.
4. Include it in foreground-pause handling and reuse shared audio/navigation.
5. Add the same in-board back-arrow button beside SCORE and wire its `onBack`
   callback to hide the session before returning Home.
6. Update the root README and verify desktop and WebAssembly builds.

## Development and validation

Run commands from the repository root, not from an individual game directory.
Use `gofmt` on changed Go files. Tests use Go's `testing` package and Fyne's
`test.NewApp`/`test.NewWindow` helpers; follow existing tests and avoid opening
real audio devices in simulation or widget tests.

Windows PowerShell examples:

```powershell
go run .
go test -ldflags="-linkmode=internal" .\...
go build -ldflags="-linkmode=internal -H=windowsgui -s -w" -o bin\game.exe .
```

For browser packaging, use the installed Fyne CLI from the repository root:

```powershell
& "$env:USERPROFILE\go\bin\fyne.exe" package -os web --release
```

Use release-mode browser builds for performance checks. `bin` and `wasm` are
generated, ignored output directories; do not hand-edit or commit their contents.
Update the root and affected game README when controls, lifecycle, build steps,
or supported games change.
