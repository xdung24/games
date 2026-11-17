package bombsweeper

import (
	"time"
)

func (g *Game) startTimer(counter *TimeCounter, stop <-chan struct{}) {
	for i := 0; i < 999; i++ {
		if !g.IsRunning {
			break
		}
		select {
		case <-stop:
			return
		default:
		}
		counter.Increment()
		select {
		case <-stop:
			return
		case <-time.After(time.Second):
		}
	}
}

func (g *Game) openTile(row, col int) {
	if g.OpenCount == 0 {
		go g.startTimer(g.TimeCounter, g.timerStop)
	}

	switch g.Tiles[row][col].Base {
	case TileTypeMine:
		g.Tiles[row][col].Open(true)
		g.lose()
	case TileType0:
		g.revealEdges(row, col)

	default:
		g.Tiles[row][col].Open(true)
		g.OpenCount++
	}
	if g.OpenCount == g.WinCount {
		g.win()
	}
}

func (g *Game) markTile(row, col int) {
	if !g.Tiles[row][col].IsFlagged {
		g.Tiles[row][col].Flag()
		g.MineCounter.Decrement()
	} else {
		g.Tiles[row][col].Unflag()
		g.MineCounter.Increment()
	}
}

func (g *Game) win() {
	g.IsRunning = false
	for itrRow := 0; itrRow < len(g.Tiles); itrRow++ {
		for itrCol := 0; itrCol < len(g.Tiles[itrRow]); itrCol++ {
			if !g.Tiles[itrRow][itrCol].IsOpen {
				g.markTile(itrRow, itrCol)
			}
		}
	}
	g.Smiley.SetState(GameStateWin)
}

func (g *Game) lose() {
	g.IsRunning = false
	for itrRow := 0; itrRow < len(g.Tiles); itrRow++ {
		for itrCol := 0; itrCol < len(g.Tiles[itrRow]); itrCol++ {
			if !g.Tiles[itrRow][itrCol].IsOpen {
				g.Tiles[itrRow][itrCol].Open(false)
			}
		}
	}
	g.Smiley.SetState(GameStateLose)
}

// 8-directional flood fill
func (g *Game) revealEdges(row, col int) {
	if row < 0 || col < 0 || row >= len(g.Tiles) || col >= len(g.Tiles[0]) {
		return
	}
	if g.Tiles[row][col].IsOpen {
		return
	}

	if g.Tiles[row][col].Base != TileType0 && g.Tiles[row][col].Base != TileTypeMine {
		g.Tiles[row][col].Open(false)
		g.OpenCount++
		return
	}
	g.Tiles[row][col].Open(false)
	g.OpenCount++
	g.revealEdges(row-1, col-1)
	g.revealEdges(row-1, col)
	g.revealEdges(row-1, col+1)
	g.revealEdges(row, col-1)
	g.revealEdges(row, col+1)
	g.revealEdges(row+1, col-1)
	g.revealEdges(row+1, col)
	g.revealEdges(row+1, col+1)
}
