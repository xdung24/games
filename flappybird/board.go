package flappybird

import (
	_ "embed"
	"fmt"
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/xdung24/games/internal/navigation"
)

//go:embed Icon.svg
var iconSVG []byte

// Icon is original artwork shared by the launcher and game.
var Icon = fyne.NewStaticResource("flappy-bird-Icon.svg", iconSVG)

//go:embed bird.svg
var birdSVG []byte

var (
	ink       = color.NRGBA{R: 32, G: 58, B: 72, A: 255}
	cream     = color.NRGBA{R: 255, G: 249, B: 232, A: 255}
	sky       = color.NRGBA{R: 152, G: 214, B: 223, A: 255}
	pipeGreen = color.NRGBA{R: 74, G: 157, B: 112, A: 255}
)

type Board struct {
	widget.BaseWidget
	game     *Game
	renderer *boardRenderer
	escape   navigation.EscapeHandler
	onBack   func()
	onMute   func()
	muted    bool
}

func NewBoard(g *Game) *Board {
	b := &Board{game: g}
	b.ExtendBaseWidget(b)
	return b
}

func (b *Board) focus() {
	if c := fyne.CurrentApp().Driver().CanvasForObject(b); c != nil {
		c.Focus(b)
	}
}

func (b *Board) activate() {
	b.escape.Reset()
	b.focus()
	b.game.Flap()
	b.Refresh()
}

func (b *Board) Tapped(*fyne.PointEvent) { b.activate() }
func (b *Board) FocusGained()            {}
func (b *Board) FocusLost()              { b.escape.Reset() }
func (b *Board) TypedRune(rune)          { b.escape.Reset() }
func (b *Board) KeyDown(*fyne.KeyEvent)  {}
func (b *Board) KeyUp(e *fyne.KeyEvent)  { b.escape.KeyUp(e.Name) }

func (b *Board) pause() {
	b.escape.Reset()
	b.game.TogglePause()
	b.focus()
	b.Refresh()
}

func (b *Board) restart() {
	b.escape.Reset()
	b.game.Restart()
	b.focus()
	b.Refresh()
}

func (b *Board) toggleMute() {
	b.escape.Reset()
	if b.onMute != nil {
		b.onMute()
	}
	b.focus()
}

func (b *Board) TypedKey(e *fyne.KeyEvent) {
	if b.escape.TypedKey(e.Name, func() {
		b.game.Pause()
		b.Refresh()
	}, b.onBack) {
		return
	}
	switch e.Name {
	case fyne.KeySpace, fyne.KeyUp:
		b.activate()
	case fyne.KeyP:
		b.pause()
	case fyne.KeyR:
		b.restart()
	case fyne.KeyM:
		b.toggleMute()
	}
}

func (b *Board) CreateRenderer() fyne.WidgetRenderer {
	r := &boardRenderer{board: b}
	r.surround = canvas.NewRectangle(ink)
	r.sky = canvas.NewRectangle(sky)
	for i := range r.clouds {
		r.clouds[i] = canvas.NewCircle(cream)
	}
	for i := range r.pipes {
		for j := range r.pipes[i] {
			r.pipes[i][j] = canvas.NewRectangle(pipeGreen)
			r.pipes[i][j].StrokeColor = ink
			r.pipes[i][j].StrokeWidth = 2
		}
	}
	r.ground = canvas.NewRectangle(color.NRGBA{R: 224, G: 200, B: 152, A: 255})
	r.grass = canvas.NewRectangle(color.NRGBA{R: 113, G: 201, B: 154, A: 255})
	for i := range r.stripes {
		r.stripes[i] = canvas.NewRectangle(pipeGreen)
	}
	r.bird = canvas.NewImageFromResource(fyne.NewStaticResource("flappy-bird.svg", birdSVG))
	r.bird.FillMode = canvas.ImageFillStretch
	r.wing = canvas.NewCircle(color.NRGBA{R: 234, G: 169, B: 91, A: 255})
	r.wing.StrokeColor, r.wing.StrokeWidth = ink, 1
	r.score = canvas.NewText("", ink)
	r.score.TextStyle.Bold = true
	r.best = canvas.NewText("", ink)
	r.veil = canvas.NewRectangle(color.NRGBA{R: 22, G: 58, B: 80, A: 210})
	r.title = canvas.NewText("", cream)
	r.title.TextStyle.Bold = true
	r.hint = canvas.NewText("", cream)
	r.detail = canvas.NewText("", cream)
	r.back = widget.NewButtonWithIcon("", theme.NavigateBackIcon(), func() {
		if b.onBack != nil {
			b.onBack()
		}
	})
	r.pause = widget.NewButtonWithIcon("", theme.MediaPauseIcon(), b.pause)
	r.restart = widget.NewButtonWithIcon("", theme.ViewRefreshIcon(), b.restart)
	r.mute = widget.NewButtonWithIcon("", theme.VolumeUpIcon(), b.toggleMute)
	b.renderer = r
	return r
}

type boardRenderer struct {
	board                      *Board
	scale, ox, oy              float32
	surround, sky              *canvas.Rectangle
	clouds                     [9]*canvas.Circle
	pipes                      [3][4]*canvas.Rectangle
	ground, grass              *canvas.Rectangle
	stripes                    [19]*canvas.Rectangle
	bird                       *canvas.Image
	wing                       *canvas.Circle
	score, best                *canvas.Text
	veil                       *canvas.Rectangle
	title, hint, detail        *canvas.Text
	back, pause, restart, mute *widget.Button
}

func (r *boardRenderer) MinSize() fyne.Size { return fyne.NewSize(320, 480) }
func (r *boardRenderer) Destroy()           {}

func (r *boardRenderer) place(o fyne.CanvasObject, x, y, w, h float32) {
	o.Move(fyne.NewPos(r.ox+x*r.scale, r.oy+y*r.scale))
	o.Resize(fyne.NewSize(w*r.scale, h*r.scale))
}

func (r *boardRenderer) placePipe(o fyne.CanvasObject, x, y, w, h float32) {
	left, right := max(0, x), min(worldWidth, x+w)
	if right <= left {
		o.Hide()
		return
	}
	o.Show()
	r.place(o, left, y, right-left, h)
}

func (r *boardRenderer) Layout(size fyne.Size) {
	r.scale = min(size.Width/worldWidth, size.Height/worldHeight)
	r.ox = (size.Width - worldWidth*r.scale) / 2
	r.oy = (size.Height - worldHeight*r.scale) / 2
	r.surround.Resize(size)
	r.Refresh()
}

func (r *boardRenderer) Objects() []fyne.CanvasObject {
	objects := []fyne.CanvasObject{r.surround, r.sky}
	for _, c := range r.clouds {
		objects = append(objects, c)
	}
	for _, pair := range r.pipes {
		for _, p := range pair {
			objects = append(objects, p)
		}
	}
	objects = append(objects, r.ground, r.grass)
	for _, stripe := range r.stripes {
		objects = append(objects, stripe)
	}
	return append(objects, r.bird, r.wing, r.veil, r.score, r.best,
		r.title, r.hint, r.detail, r.back, r.pause, r.restart, r.mute)
}

func (r *boardRenderer) centered(t *canvas.Text, y float32) {
	t.Refresh()
	t.Move(fyne.NewPos(r.ox+(worldWidth*r.scale-t.MinSize().Width)/2, r.oy+y*r.scale))
}

func (r *boardRenderer) Refresh() {
	if r.scale <= 0 {
		return
	}
	g := r.board.game
	r.place(r.sky, 0, 0, worldWidth, worldHeight)
	for i, c := range r.clouds {
		x := float32(i/3)*130 + float32(i%3)*17 - 15
		y := float32(95+(i/3%2)*45) - float32(i%3%2)*10
		r.place(c, x, y, 42, 30)
	}
	for i, p := range g.pipes {
		parts := r.pipes[i]
		r.placePipe(parts[0], p.x+4, 0, pipeWidth-8, p.gapTop-18)
		r.placePipe(parts[1], p.x, p.gapTop-18, pipeWidth, 18)
		r.placePipe(parts[2], p.x+4, p.gapTop+pipeGap+18, pipeWidth-8, groundY-p.gapTop-pipeGap-18)
		r.placePipe(parts[3], p.x, p.gapTop+pipeGap, pipeWidth, 18)
	}
	r.place(r.ground, 0, groundY, worldWidth, worldHeight-groundY)
	r.place(r.grass, 0, groundY, worldWidth, 12)
	offset := float32(math.Mod(float64(g.travel), 20))
	for i, stripe := range r.stripes {
		x := float32(i)*20 - offset
		left, right := max(0, x), min(worldWidth, x+10)
		r.place(stripe, left, groundY+3, max(0, right-left), 4)
	}
	r.place(r.bird, birdX-21, g.birdY-18, 48, 36)
	wingLift := float32(math.Sin(float64(g.travel)*0.16)) * 4
	if g.state != StatePlaying {
		wingLift = 0
	}
	r.place(r.wing, birdX-16, g.birdY+wingLift, 21, 12)
	r.score.Text = fmt.Sprintf("SCORE  %d", g.score)
	r.score.TextSize = 23 * r.scale
	r.centered(r.score, 14)
	r.best.Text = fmt.Sprintf("BEST  %d", g.best)
	r.best.TextSize = 12 * r.scale
	r.centered(r.best, 45)

	r.place(r.veil, 0, 150, worldWidth, 195)
	r.title.TextSize, r.hint.TextSize, r.detail.TextSize = 30*r.scale, 16*r.scale, 13*r.scale
	switch g.state {
	case StateReady:
		r.title.Text = "FLAPPY BIRD"
		r.hint.Text = "Tap or press Space to flap"
		r.detail.Text = "Fly through the gaps. Don't hit a pipe!"
	case StatePaused:
		r.title.Text = "PAUSED"
		r.hint.Text = "Tap or press Space to resume"
		r.detail.Text = "Esc again returns Home"
	case StateGameOver:
		r.title.Text = "NICE FLIGHT!"
		r.hint.Text = fmt.Sprintf("Score %d  /  Best %d", g.score, g.best)
		r.detail.Text = "Tap or press Space to try again"
	}
	for _, o := range []fyne.CanvasObject{r.veil, r.title, r.hint, r.detail} {
		if g.state == StatePlaying {
			o.Hide()
		} else {
			o.Show()
		}
	}
	r.centered(r.title, 180)
	r.centered(r.hint, 245)
	r.centered(r.detail, 282)

	buttons := []*widget.Button{r.back, r.pause, r.restart, r.mute}
	for i, button := range buttons {
		size := button.MinSize()
		button.Resize(size)
		button.Move(fyne.NewPos(r.ox+(25+float32(i)*87)*r.scale,
			r.oy+worldHeight*r.scale-size.Height-8*r.scale))
	}
	pauseIcon := theme.MediaPauseIcon()
	if g.state == StatePaused {
		pauseIcon = theme.MediaPlayIcon()
	}
	if r.pause.Icon != pauseIcon {
		r.pause.SetIcon(pauseIcon)
	}
	muteIcon := theme.VolumeUpIcon()
	if r.board.muted {
		muteIcon = theme.VolumeMuteIcon()
	}
	if r.mute.Icon != muteIcon {
		r.mute.SetIcon(muteIcon)
	}
	canPause := g.state == StatePlaying || g.state == StatePaused
	if canPause && r.pause.Disabled() {
		r.pause.Enable()
	} else if !canPause && !r.pause.Disabled() {
		r.pause.Disable()
	}
	canvas.Refresh(r.board)
}
