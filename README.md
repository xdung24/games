# game

![game](https://img.shields.io/badge/Fyne-v2.7-blue)

## Play

```sh
go run .
```

The app opens on a game-selection menu with up to 10 games per page. Use
**Previous** and **Next** to change pages, then tap anywhere on a game card to
open it. Brick Breaker and Star Breaker are available. Select the small
back arrow beside **SCORE** in either game to
return to the menu; each game is paused and kept for when you return.
You can also press **Esc** to pause, then press **Esc** again to return Home.
Other input cancels this two-press sequence.
This menu is available in both desktop and browser builds.
The Home menu displays the root `Icon.png` above its subtitle, and each game
card displays its game's `Icon.png` to the left of the title and description.
These icons are embedded in the app, so no separate image files are needed
when running a built binary.
Game cards smoothly highlight their background and border on mouse hover or
keyboard focus. Use **Up/Down** to select a game, **Left/Right** to change pages,
and **Enter** or **Space** to play. **Tab** also moves keyboard focus.
The selected card stays visible and is restored when you return to the menu.

The root `main` package owns the launcher and menu. The `brickbreaker` and
`starbreaker` directories are importable game packages; each owns its rendering,
audio, and simulation loop. They share one audio device through `internal/audio`.
The launcher uses `NewScreen`, `Show`, and `Pause` to manage each game without
accessing its internal state. Run build and package commands from the repository root.

To build a standalone binary:

```sh
go build -o game .
./game
```

On Windows, use Go's internal linker:

```powershell
go build -ldflags="-linkmode=internal -H=windowsgui -s -w" -o bin\game.exe .
```

## Play in a browser

From the project directory in Windows PowerShell, install the Fyne CLI and
start its WebAssembly development server:

```powershell
$env:GOBIN = Join-Path $env:USERPROFILE "go\bin"
go install fyne.io/tools/cmd/fyne@latest
& "$env:GOBIN\fyne.exe" serve
```

Open the address printed by the server (normally http://localhost:8080) in a
modern browser. Press `Ctrl+C` in PowerShell to stop the server. It listens on
all network interfaces, so use a trusted network or restrict access with your
firewall.

To create files for a static web host instead:

```powershell
& "$env:GOBIN\fyne.exe" package -os web --release
```

Use `--release` for gameplay: without it, Fyne wraps WebGL calls in debug
checks that can substantially reduce frame rate. `fyne serve` uses release mode
by default; avoid `fyne serve --debug` when measuring performance.

To use the included `index.html` entry page, build the package above, then serve
the project directory locally (requires Python):

```powershell
python -m http.server 8080 --bind 127.0.0.1
```

Open http://127.0.0.1:8080/index.html. The page loads the generated game from
`wasm/index.html`. Re-run the package command after changing the Go code.
The generated `wasm` directory is ignored by Git.
After rebuilding an older debug package, reload the browser with `Ctrl+F5`
to avoid cached assets. Keep browser hardware acceleration enabled.

Upload the generated `wasm` directory contents to your host and serve them over
HTTP(S); do not open the HTML file directly. Configure the host to serve
`.wasm` files as `application/wasm`. Both commands may increment the build
number in `FyneApp.toml`.

Alternatively, deploy `index.html` alongside the `wasm` directory, keeping that
directory structure, and include `Icon.png` for the page icon.

Browsers require an interaction before audio starts. The current game may
remain silent if the audio device is not ready during startup.
Use the mouse or arrow keys to move and click or press `Space` to launch,
pause/resume, or play again. The current Fyne web driver does not deliver the
letter-key shortcuts (`A`/`D`, `P`, `R`, `M`) used by the desktop build.
