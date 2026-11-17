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

The sky and ground are cached, and frames reuse an opaque RGBA buffer that Fyne
can upload without allocating and converting another image. Browser rendering
is capped at the logical 400x600 resolution (preserving the viewport's aspect
ratio), then smoothly scaled by the GPU; text and controls remain full resolution.
Native builds retain full-resolution rendering. Physics, update timing, and input
behavior are unchanged.
Use the release WebAssembly build described in the root README for gameplay.
