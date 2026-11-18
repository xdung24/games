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
- [brickbreaker/](../brickbreaker) and [asteroidshooter/](../asteroidshooter): importable
  game packages, not standalone applications.
  - `game.go`: simulation, state transitions, physics, and sound-event callbacks.
  - `board.go`: Fyne widget, input handling, HUD, and canvas renderer.
  - `screen.go`: session ownership, navigation, focus, and simulation loop.
  - `sound.go`: synthesized effects and per-game mute handling.
  - `theme.go`: game-specific theme.
  - Asteroid Shooter's `shaders.go`: desktop and OpenGL ES background shaders.
- [flappychick/](../flappychick): reference for lightweight retained rendering,
  elapsed-time simulation, and skipping redraws outside active play.
- [flappygophy/](../flappygophy): simulation/input in `main.go`, reusable visual
  objects in `scene.go`, and session/HUD management in `screen.go`.
- [internal/audio/](../internal/audio): the shared Oto audio context.
- [internal/navigation/](../internal/navigation): shared double-Esc handling.
- [internal/touchcontrols/](../internal/touchcontrols): Go/Fyne SVG buttons,
  held-touch tracking, game-screen layout, and platform pointer translation.
- [internal/sprite/](../internal/sprite): sprite-sheet engine that loads
  `cmd/spritesheet` JSON + image, looks up frames by name, plays clips, and
  draws scaled/mirrored frames.
- [cmd/spritesheet/](../cmd/spritesheet): CLI that slices grid sprite sheets
  into a transparent PNG/WebP atlas plus JSON frame metadata.
- [index.html](../index.html): full-window iframe loading `web/index.html`.
- [web/](../web): checked-in loading shell and browser-only zoom/keyboard
  protections. `mobile.js` must not implement gameplay; only WASM is generated.
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
- Simulation loops target 60 FPS. For elapsed-time models, clamp elapsed time
  to 0.05 seconds; preserve existing fixed-step timing where used.
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
- Display the root `Icon.png` above the Home subtitle and each game's icon
  to the left of its card's title and description. Embed launcher icons with
  `go:embed` and preserve their aspect ratios.
- Preserve existing back-arrow placement: beside SCORE in the breaker games
  and bottom-left in the flappy games. For the breaker games, use the small,
  low-importance back-arrow button beside SCORE.
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

## Rendering and performance requirements

- Treat visual quality and responsiveness as joint requirements. Do not solve
  slowness by lowering the full-scene resolution, blurring/pixelating artwork,
  removing detail, or dropping the frame rate without explicit user approval.
- Prefer retained scene objects, following
  [Flappy Chick's renderer](../flappychick/board.go) and
  [Flappy Gophy's scene](../flappygophy/scene.go). Create objects once, then
  move them and toggle visibility. Do not rebuild the scene on every tick.
- Do not paint or allocate full-screen CPU images every frame. Avoid per-frame
  pixel loops, image-format conversions, and full-screen texture uploads.
  Use GPU-drawn primitives/gradients and cached artwork instead.
- Keep static layers static. Generate detailed textures only when their size
  or content changes; do not call `Refresh()` on an unchanged raster or image.
  Moving a sprite should not invalidate its texture.
- Render SVG artwork at native display density and cache animation poses.
  Validate centering, transparent margins, and clipping in Fyne itself, not
  just an SVG preview. Never trade away high-DPI sharpness as a shortcut.
- Clip sprites at the playfield boundary, for example with `container.NewClip`,
  rather than changing their dimensions every frame as they enter or leave
  the screen. Resizing SVG images can trigger expensive rasterization.
- Avoid unnecessary allocations in update/render loops. Reuse bounded object
  pools, buffers, and model slices where safe. Do not synthesize or decode
  reusable assets repeatedly in the hot path.
- Redraw only when something changes. Skip hidden games, and stop continuous
  rendering in ready/title, paused, and game-over states unless an intentional
  visible animation requires it. Always draw the final collision/state-change
  frame. Idle games should not consume an active game's CPU/GPU budget.
- Keep physics independent of painting and device pixel ratio. Do not change
  speed, collision geometry, scoring, or input timing to hide rendering stalls.
  Avoid unbounded update backlogs or catch-up work after backgrounding.

## Mobile experience requirements

- Implement game controls in Go, inside each game's Fyne screen/board. Reuse
  `internal/touchcontrols` for SVG button rendering, held touches, and layout.
  Bind controls directly to Go actions/model input, never to synthetic key events.
- Keep HTML/JavaScript restricted to loading WASM and browser configuration
  (viewport, zoom protection, and suppressing the on-screen keyboard).
  Never create gameplay DOM buttons, track game input/state, or dispatch game
  keyboard events in `mobile.js` or other browser-shell assets.
- If a Fyne platform driver lacks multi-touch dispatch, keep the minimal pointer
  adapter in Go behind build tags. Route pointer IDs to the actual Go widgets;
  do not duplicate widget geometry or game behavior in JavaScript.
- Keep all games playable without a physical keyboard. Use tap/drag when that
  already provides complete control; add virtual buttons only for otherwise
  inaccessible actions, as in Asteroid Shooter.
- Support simultaneous held actions such as steering, thrust, and firing through
  Go widget touch IDs and pointer capture where needed. Releasing one finger
  must not release another finger or a held physical keyboard action.
- Release held controls on pointer-up, cancellation, lost capture, blur,
  visibility loss, and navigation. Pause through the existing lifecycle.
  Prevent touch controls from also activating the game underneath them.
- Keep touch targets at least 44 CSS pixels where practical. Respect safe
  areas, reserve playfield space for control bars, and avoid obscuring the
  player, HUD, pause, or Home controls in portrait and landscape.
- Preserve the viewport/gesture restrictions in both HTML entry pages.
  Keep the Fyne dummy input read-only with `inputmode="none"` so focusing
  game controls does not open the on-screen keyboard or resize the viewport.
  Do not disable physical keyboard input.
- Test narrow phone layouts, landscape, touch dragging, rapid taps, multi-touch,
  pause/resume, and returning Home. Avoid accidental zoom, page scrolling,
  context menus, duplicate actions, and stuck controls.
- Test Go control widgets and game action bindings with Fyne's headless helpers,
  then validate the release WASM with real browser touches. Verify direct system
  actions (pause/restart/mute), not just delivery of keyboard events.

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
- Edit the checked-in root/browser entry pages and assets in `web`, not the
  generated WASM binary. Preserve the iframe's relative URL, autoplay
  permission, and mobile input protections. Do not replace the custom web
  shell with Fyne-generated HTML without retaining these integrations.

## Sprite sheets

- Generate raster sprite sheets with [cmd/spritesheet](../cmd/spritesheet)
  and load them with `internal/sprite`. Do not hard-code grid sizes, row
  indexes or frame counts in game code, and do not scan pixels at runtime to
  find sprite bounds; use the JSON's frame names, animation lists and `trim`.
- Embed **JSON + WebP** (lossless) in a game's `assets/`. WebP is about 40%
  smaller than PNG with identical pixels and decodes no slower; embedded
  images go straight into the WASM download and gzip barely shrinks either
  format. Do not embed a PNG copy of the same sheet.
- Keep PNG copies and original source sheets under `resources/` for editing.
  `sprite.Load` uses the first image in the JSON's `images` list that exists,
  so the embedded PNG can be omitted without code changes.
- Select animations by name (for example `sheet.Clip("walk", 0, 0, 0.085,
  true)`) and stand sprites on the floor with `Frame.BottomPadding()`.
  Load sheets lazily, as Kung Fu Master does with `sync.OnceValue`, so other
  games do not pay the decode cost at launcher startup.
- Document the exact `cmd/spritesheet` command used to regenerate a game's
  sheet in that game's README.

## Adding a game

1. Add an importable package with the model/board/screen structure above,
   an `Icon.png`, a README describing controls, and focused tests.
2. Embed its icon and add a `gameEntry` in `main.go`.
3. Lazily create and retain its screen, select its theme on launch, and pass the
   shared Home callback to `NewScreen`.
4. Include it in foreground-pause handling and reuse shared audio/navigation.
5. Reuse the appropriate in-game back-arrow pattern and wire its `onBack`
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
go test -tags "ci migrated_fynedo" .\flappygophy
go build -tags "release no_emoji migrated_fynedo" -ldflags="-linkmode=internal -H=windowsgui -s -w" -o bin\games.exe .
```

Select the changed packages for tests; use `.\...` when cross-package behavior
requires the full suite. The `ci` tag enables headless Fyne validation.
For a release browser build matching [Netlify](../netlify.toml):

```powershell
$env:GOOS = "js"
$env:GOARCH = "wasm"
go build -tags "release no_emoji migrated_fynedo" -ldflags="-s -w" -o web\games.wasm .
Remove-Item Env:GOOS, Env:GOARCH
```

Use release-mode browser builds for performance checks; debug WebGL checks
can distort results. Preserve `migrated_fynedo`, which matches the app's Fyne
threading migration. Do not hand-edit or commit generated `bin` outputs or
`web/games.wasm`.
Update the root and affected game README when controls, lifecycle, build steps,
or supported games change.

## Performance verification before declaring a fix

1. Reproduce and measure the actual slow path before changing it. Compare with
   Flappy Chick using the same build, viewport, device pixel ratio, focus,
   visibility, and CPU-throttling settings. Verify gameplay is active rather
   than paused; background tabs throttle timers and animation frames.
2. Aim for 60 FPS (about 16.7 ms per frame) on representative mobile hardware.
   Record frame-time distribution, long stalls, main-thread work, allocations,
   and texture uploads as relevant. Report browser/tool limitations instead
   of claiming an unverified frame rate.
3. Measure both warmed-up gameplay and cold paths: opening a game, first flap,
   new pipes/assets, resize/orientation changes, and background/resume. Do not
   hide startup or spawn stalls inside a favorable average.
4. Verify idle screens stop rendering and steady movement does not repaint or
   upload a full-screen bitmap. Microbenchmarks alone do not prove smooth
   gameplay; run the rebuilt WASM in a browser and confirm the asset is fresh,
   using a hard reload when needed.
5. Inspect before/after screenshots at normal and high density, including a
   3x-DPI phone and landscape. Check crisp artwork, correct proportions,
   unclipped sprites, readable HUD text, and reachable controls. A smaller
   raster or a passing benchmark is not an acceptable proxy for visual quality.
6. Add focused regressions for the fixed behavior, run affected tests and the
   release WASM build, and report measured results and remaining limitations.
