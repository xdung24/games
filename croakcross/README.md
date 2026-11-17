# CroakCross

A nostalgic, single-player **Frogger** clone built with [Fyne](https://fyne.io).
Hop across five lanes of traffic, over the median, then ride the logs and
turtles across the river to the five home bays at the top. Fill every bay to
clear the level - the next one runs faster and more turtles dive. The game ends
when your frogs run out.

![CroakCross](https://img.shields.io/badge/Fyne-v2.8-blue)

## Play

```sh
go run .
```

CroakCross is also available from the main game collection on desktop, mobile,
and web.

To build a standalone binary:

```sh
go build -o croakcross .
./croakcross
```

## Controls

| Input | Action |
| --- | --- |
| `←` `↑` `→` `↓` / `W` `A` `S` `D` | Hop one cell |
| Tap (in the direction you want, relative to the frog) / swipe | Hop on a touch screen |
| `Space` / `Enter` / tap | Start · pause / resume · play again |
| `P` | Pause / resume |
| `R` | Restart |
| `M` | Mute / unmute the sound |
| `Esc` (twice) / Home button | Pause, then return to the game menu |

## How it plays

- **Three frogs.** Lose one to a car, the river, the clock, or a bad landing
  on the top bank. At zero, it's game over. An extra frog arrives every 10,000 points.
- **The road.** Cars, tractors, racers and trucks run in alternating directions.
  Touch one and you're flat.
- **The river.** Open water is fatal - stand on a log or a turtle and ride it.
  Some turtle groups dive; hop off before they go under, and don't get carried
  off the edge of the screen.
- **Home.** Only the five bays are safe. Land between them, or in a bay you've
  already filled, and you splat. A fly sometimes rests in a bay for a 200-point
  bonus; from level 3 a crocodile sometimes lurks there instead.
- **The clock.** Each frog has 30 seconds to get home. Every forward hop is
  worth 10, reaching home 50 plus 10 per second left on the clock, and filling
  all five bays 1,000 before the next level starts.
- **Ten levels.** Each one runs the traffic faster and sends more turtles
  diving. Cross all ten and you win.
