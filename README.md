# Games

![Games](https://img.shields.io/badge/Fyne-v2.7-blue)

Choose a game to see a brief loading screen with its icon and an animated
indicator before it opens. The indicator respects the system animation setting.
Returning Home preserves your game progress and menu selection.

## Play in desktop

To build a standalone binary:

```sh
go build  -tags 'release no_emoji migrated_fynedo' -ldflags='-linkmode=internal -H=windowsgui -s -w' -o bin/games.exe .
```

## Play in a browser

```sh
GOOS=js GOARCH=wasm go build -tags 'release no_emoji migrated_fynedo' -ldflags='-s -w' -o web/games.wasm .
```

Open http://127.0.0.1:8080/web/index.html. The page loads the game from
`web/index.html`. Re-run the binary build after changing the Go code.
The generated `web/games.wasm` binary is ignored by Git.
After rebuilding an older debug package, reload the browser with `Ctrl+F5`
to avoid cached assets. Keep browser hardware acceleration enabled.

## Deploy to web

Upload the generated `wasm` directory contents to your host and serve them over
HTTP(S); do not open the HTML file directly. Configure the host to serve
`.wasm` files as `application/wasm`.

Netlify uses `netlify.toml` to build the app and publish the checked-in `web`
shell, including the mobile controls described above.

Alternatively, deploy `index.html` alongside the `web` directory, keeping that
directory structure, and include `Icon.png` for the page icon.
