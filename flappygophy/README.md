# Flappy Gophy

A simple game where you aim to guide a hyperactive Gopher through some pipes.

## Play

Run `go run .` from the repository root and select **Flappy Gophy**.
This directory is an importable game package, not a standalone executable.

Click/tap or press **Space**, **Enter**, or **Up** to start, flap, or retry.
Use the bottom-right button to pause/resume and the bottom-left arrow for Home.
Press **Esc** to pause, then **Esc** again to return Home. A paused round is
preserved when you return; click/tap or press a flap key to resume it.

Best scores are saved when a round ends. Audio uses the launcher's shared device.

## Web performance

Like Flappy Chick, the game moves reusable scene objects rather than repainting
a full-screen bitmap. Gopher poses and clouds use cached SVG artwork rendered
at the display's resolution. The sky and pipes use gradients, and the detailed
ground texture is generated only when its size changes. There is no low-resolution
web rendering cap. Physics, update timing, and input behavior are unchanged.
Like Flappy Chick, title, paused, and game-over screens stop continuous rendering
until input or resizing requires an update. Clouds move during active play.
Use the release WebAssembly build described in the root README for gameplay.
