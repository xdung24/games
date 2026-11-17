package acornrun

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
	"github.com/xdung24/games/internal/navigation"
)

const swipeThreshold = 20

type Board struct {
	widget.BaseWidget
	game         *Game
	renderer     *boardRenderer
	input        Input
	jumpPulse    bool
	dragX        float32
	dragY        float32
	touchDir     int
	touchTill    time.Time
	escape       navigation.EscapeHandler
	onBack       func()
	onToggleMute func()
}

func NewBoard(game *Game) *Board {
	board := &Board{game: game}
	board.ExtendBaseWidget(board)
	return board
}

func (b *Board) CreateRenderer() fyne.WidgetRenderer {
	r := &boardRenderer{
		board:    b,
		raster:   canvas.NewRaster(func(width, height int) image.Image { return b.game.renderImage(width, height) }),
		veil:     canvas.NewRectangle(color.NRGBA{R: 13, G: 12, B: 24, A: 190}),
		title:    canvas.NewText("", color.NRGBA{R: 255, G: 191, B: 107, A: 255}),
		subtitle: canvas.NewText("", color.NRGBA{R: 243, G: 237, B: 255, A: 255}),
		hud:      canvas.NewText("", color.NRGBA{R: 225, G: 246, B: 239, A: 255}),
	}
	r.raster.ScaleMode = canvas.ImageScalePixels
	r.title.TextStyle.Bold = true
	r.title.Alignment = fyne.TextAlignCenter
	r.subtitle.Alignment = fyne.TextAlignCenter
	r.hud.TextStyle.Bold = true
	b.renderer = r
	return r
}

func (b *Board) Refresh() {
	b.BaseWidget.Refresh()
}

func (b *Board) FocusGained() {}

func (b *Board) FocusLost() {
	b.input = Input{}
	b.jumpPulse = false
	b.escape.Reset()
}

func (b *Board) TypedRune(rune) { b.escape.Reset() }

func (b *Board) TypedKey(e *fyne.KeyEvent) {
	if b.escape.TypedKey(e.Name, func() {
		b.game.Pause()
		b.Refresh()
	}, b.onBack) {
		return
	}
	switch e.Name {
	case fyne.KeySpace:
		if b.game.state == StateReady || b.game.state == StatePaused || b.game.state == StateGameOver || b.game.state == StateWon {
			b.activate()
		}
	case fyne.KeyP:
		b.game.TogglePause()
	case fyne.KeyR:
		b.game.Restart()
	case fyne.KeyM:
		if b.onToggleMute != nil {
			b.onToggleMute()
		}
	}
	b.Refresh()
}

func (b *Board) KeyDown(e *fyne.KeyEvent) {
	switch e.Name {
	case fyne.KeyUp, fyne.KeyW, fyne.KeyX, fyne.KeySpace:
		b.input.Jump = true
	case fyne.KeyLeft, fyne.KeyA:
		b.input.Left = true
	case fyne.KeyRight, fyne.KeyD:
		b.input.Right = true
	case fyne.KeyZ:
		b.input.Run = true
	}
}

func (b *Board) KeyUp(e *fyne.KeyEvent) {
	b.escape.KeyUp(e.Name)
	switch e.Name {
	case fyne.KeyUp, fyne.KeyW, fyne.KeyX, fyne.KeySpace:
		b.input.Jump = false
	case fyne.KeyLeft, fyne.KeyA:
		b.input.Left = false
	case fyne.KeyRight, fyne.KeyD:
		b.input.Right = false
	case fyne.KeyZ:
		b.input.Run = false
	}
}

func (b *Board) Tapped(*fyne.PointEvent) {
	b.escape.Reset()
	switch b.game.state {
	case StateReady:
		b.game.Start()
		b.jumpPulse = true
	case StatePlaying:
		b.jumpPulse = true
	case StatePaused:
		b.game.TogglePause()
	case StateGameOver, StateWon:
		b.game.Restart()
	}
	b.Refresh()
}

func (b *Board) Dragged(e *fyne.DragEvent) {
	b.escape.Reset()
	b.dragX += e.Dragged.DX
	b.dragY += e.Dragged.DY
}

func (b *Board) DragEnd() {
	dx, dy := b.dragX, b.dragY
	b.dragX, b.dragY = 0, 0
	if abs(dx) < swipeThreshold && abs(dy) < swipeThreshold {
		return
	}
	if abs(dx) > abs(dy) {
		b.touchDir = 1
		if dx < 0 {
			b.touchDir = -1
		}
		b.touchTill = time.Now().Add(220 * time.Millisecond)
	} else if dy < 0 {
		b.jumpPulse = true
	}
}

func (b *Board) activate() {
	switch b.game.state {
	case StateReady:
		b.game.Start()
	case StatePlaying, StatePaused:
		b.game.TogglePause()
	case StateGameOver, StateWon:
		b.game.Restart()
	}
}

func (b *Board) frameInput() Input {
	input := b.input
	if time.Now().Before(b.touchTill) {
		input.Left = input.Left || b.touchDir < 0
		input.Right = input.Right || b.touchDir > 0
	}
	input.Jump = input.Jump || b.jumpPulse
	b.jumpPulse = false
	return input
}

type boardRenderer struct {
	board    *Board
	raster   *canvas.Raster
	veil     *canvas.Rectangle
	title    *canvas.Text
	subtitle *canvas.Text
	hud      *canvas.Text
}

func (r *boardRenderer) MinSize() fyne.Size {
	return fyne.NewSize(320, 300)
}

func (r *boardRenderer) Layout(size fyne.Size) {
	r.raster.Move(fyne.NewPos(0, 0))
	r.raster.Resize(size)
	r.veil.Move(fyne.NewPos(0, 0))
	r.veil.Resize(size)
	r.hud.TextSize = max(11, min(18, size.Width/42))
	r.hud.Move(fyne.NewPos(10, 8))
	r.hud.Resize(fyne.NewSize(size.Width-20, r.hud.TextSize*1.5))
	r.title.TextSize = max(22, min(48, size.Width/12))
	r.title.Resize(fyne.NewSize(size.Width-24, r.title.TextSize*1.5))
	r.title.Move(fyne.NewPos(12, size.Height*0.39))
	r.subtitle.TextSize = max(12, min(20, size.Width/28))
	r.subtitle.Resize(fyne.NewSize(size.Width-24, r.subtitle.TextSize*2))
	r.subtitle.Move(fyne.NewPos(12, size.Height*0.54))
	r.Refresh()
}

func (r *boardRenderer) Refresh() {
	r.raster.Refresh()
	r.hud.Text = fmt.Sprintf("ACORNS %d/%d    SCORE %d    LIVES %d",
		r.board.game.acornCount, len(r.board.game.acorns), r.board.game.score, r.board.game.lives)
	canvas.Refresh(r.hud)

	title, subtitle := "", ""
	switch r.board.game.state {
	case StateReady:
		title, subtitle = "ACORN RUN", "Arrows/A,D move · Space/X jump · swipe to move"
	case StatePaused:
		title, subtitle = "Paused", "Space resumes · Esc twice returns Home"
	case StateDying:
		title, subtitle = "Careful!", "A beetle got you · keep going"
	case StateGameOver:
		title, subtitle = "Game Over", "Space or tap to try again"
	case StateWon:
		title = "Woodland cleared!"
		subtitle = fmt.Sprintf("%d acorns · %d points · Space to play again",
			r.board.game.acornCount, r.board.game.score)
	}
	if title == "" {
		r.veil.Hide()
		r.title.Hide()
		r.subtitle.Hide()
		return
	}
	r.veil.Show()
	r.title.Text = title
	r.subtitle.Text = subtitle
	r.title.Show()
	r.subtitle.Show()
	canvas.Refresh(r.veil)
	canvas.Refresh(r.title)
	canvas.Refresh(r.subtitle)
}

func (r *boardRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.raster, r.veil, r.hud, r.title, r.subtitle}
}

func (r *boardRenderer) Destroy() {}
