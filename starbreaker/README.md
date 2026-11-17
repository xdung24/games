# Starbreaker

A neon vector take on the arcade classic **Asteroids**, built with [Fyne](https://fyne.io).
Drift through a glowing nebula, blast the rocks into ever-smaller pieces, dodge the
hunting saucer, and clear wave after escalating wave.

## Controls

| Input | Action |
| --- | --- |
| `←` `→` / `A` `D` | Rotate the ship |
| `↑` / `W` | Thrust (momentum carries you) |
| `Space` | Fire · start · play again |
| `H` / `↓` | Hyperspace — jump to a random spot (no protection!) |
| `P` | Pause / resume |
| `Esc`, then `Esc` again | Pause, then return to the Home menu |
| `R` | Restart |
| `M` | Mute / unmute (remembered between runs) |

A tap or click also starts the game and fires.
The custom browser build adds SVG-icon multi-touch controls on touch-capable
devices. Left and Right sit at the bottom-left, Jump sits at the bottom-center,
and Thrust and Fire sit at the bottom-right. Hold steering, thrust,
and fire simultaneously to steer and shoot. Pause, Restart, and Mute form a
separate top-right group below the HUD. Icons retain accessible names and tooltips.
Select the small back arrow beside **SCORE** to return to the launcher.
Your session is paused and preserved;
opening Star Breaker again restores it. Run `go run .` from the repository root,
not from this directory.

## How it plays

- **Three ships.** Lose one whenever a rock, the saucer or a saucer shot catches you.
  A fresh ship blinks while it's briefly invulnerable. Out of ships, it's game over.
- **Split the rocks.** A large rock breaks into two mediums, a medium into two smalls,
  and a small shatters for good. Smaller rocks are worth more (20 / 50 / 100) and move
  faster — clearing the field is a race against the chaos you create.
- **The saucer.** A hunter crosses the field now and then. The big one sprays wildly
  (200 pts); the rare small one aims true (1000 pts).
- **Bonus ship** every 10,000 points.
- **Rising waves.** Each cleared field spawns a larger, faster one.

The whole world wraps: fly off any edge and you reappear on the opposite side, and
so do your shots and the rocks.

## Rendering and mobile performance

The renderer reuses neon polygons, tracers, particle objects, and vertex buffers
instead of rebuilding the scene each frame. HUD text changes only when needed.
The native-resolution nebula and vector detail are unchanged; one game clock
drives both gameplay and the background. Pausing, game over, or leaving the game
stops animation, while the attract screen still has drifting rocks and stars.

Run the renderer allocation benchmark with:

```sh
go test -tags "ci migrated_fynedo" ./starbreaker -run '^$' -bench BenchmarkBoardRefresh -benchmem
```

The benchmark covers 5 or 30 rocks plus 80 particles. It measures scene updates,
not GPU painting or browser FPS; verify active gameplay in a foreground release
browser at phone dimensions and high DPI as well.

Run the browser touch-input regressions with `node --test web/mobile.test.cjs`.
