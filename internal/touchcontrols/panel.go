package touchcontrols

import (
	"image/color"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

const Height float32 = 112

type target struct {
	button *Button
	pos    fyne.Position
	size   fyne.Size
}

// Panel is the game screen root. Controls can float over the playfield or
// occupy a reserved footer, with optional system buttons below the game's HUD.
type Panel struct {
	widget.BaseWidget
	content                     fyne.CanvasObject
	Left, Center, Right, System []*Button
	Overlay                     bool
	// ControlsY is the vertical center as a fraction of panel height; zero keeps bottom alignment.
	ControlsY float32
	mu        sync.RWMutex
	targets   []target
	size      fyne.Size
	detach    func()
}

func NewPanel(content fyne.CanvasObject, left, center, right, system []*Button) *Panel {
	p := &Panel{content: content, Left: left, Center: center, Right: right, System: system}
	p.ExtendBaseWidget(p)
	return p
}

func (p *Panel) Reset() {
	for _, group := range [][]*Button{p.Left, p.Center, p.Right, p.System} {
		for _, button := range group {
			button.Reset()
		}
	}
}

func (p *Panel) Attach() {
	p.Detach()
	p.detach = attach(p)
}

func (p *Panel) Detach() {
	if p.detach != nil {
		p.detach()
		p.detach = nil
	}
	p.Reset()
}

func (p *Panel) hit(x, y, width, height float32) *Button {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if width <= 0 || height <= 0 {
		return nil
	}
	pos := fyne.NewPos(x*p.size.Width/width, y*p.size.Height/height)
	for _, t := range p.targets {
		if pos.X >= t.pos.X && pos.Y >= t.pos.Y &&
			pos.X < t.pos.X+t.size.Width && pos.Y < t.pos.Y+t.size.Height {
			return t.button
		}
	}
	return nil
}

func (p *Panel) CreateRenderer() fyne.WidgetRenderer {
	footer := canvas.NewRectangle(color.NRGBA{0x0B, 0x1B, 0x2B, 0xFF})
	objects := []fyne.CanvasObject{p.content, footer}
	for _, group := range [][]*Button{p.Left, p.Center, p.Right, p.System} {
		for _, b := range group {
			objects = append(objects, b)
		}
	}
	return &panelRenderer{panel: p, footer: footer, objects: objects}
}

type panelRenderer struct {
	panel   *Panel
	footer  *canvas.Rectangle
	objects []fyne.CanvasObject
}

func (r *panelRenderer) Layout(size fyne.Size) {
	p := r.panel
	fieldHeight := max(0, size.Height-Height)
	if p.Overlay {
		fieldHeight = size.Height
		r.footer.Hide()
	} else {
		r.footer.Show()
	}
	p.content.Move(fyne.NewPos(0, 0))
	p.content.Resize(fyne.NewSize(size.Width, fieldHeight))
	r.footer.Move(fyne.NewPos(0, fieldHeight))
	r.footer.Resize(fyne.NewSize(size.Width, Height))
	buttonSize := max(float32(44), min(float32(56), size.Width*0.12))
	y := size.Height - (Height+buttonSize)/2
	if p.Overlay && p.ControlsY > 0 {
		y = max(0, min(size.Height-buttonSize, size.Height*p.ControlsY-buttonSize/2))
	}
	targets := make([]target, 0, len(p.Left)+len(p.Center)+len(p.Right)+len(p.System))
	placeButton := func(b *Button, x, y, side float32) {
		pos, s := fyne.NewPos(x, y), fyne.NewSize(side, side)
		b.Move(pos)
		b.Resize(s)
		targets = append(targets, target{b, pos, s})
	}
	place := func(group []*Button, x, y, side float32) {
		for _, b := range group {
			placeButton(b, x, y, side)
			x += side + 6
		}
	}
	groupWidth := func(group []*Button) float32 {
		if len(group) == 0 {
			return 0
		}
		return float32(len(group))*buttonSize + float32(len(group)-1)*6
	}
	place(p.Left, 8, y, buttonSize)
	place(p.Center, (size.Width-groupWidth(p.Center))/2, y, buttonSize)
	place(p.Right, size.Width-8-groupWidth(p.Right), y, buttonSize)
	systemY := float32(math.Max(64, float64(size.Height*0.08)))
	for i, b := range p.System {
		placeButton(b, size.Width-52, systemY+float32(i)*50, 44)
	}
	p.mu.Lock()
	p.targets, p.size = targets, size
	p.mu.Unlock()
}

func (r *panelRenderer) MinSize() fyne.Size           { return fyne.NewSize(320, 240) }
func (r *panelRenderer) Destroy()                     { r.panel.Detach() }
func (r *panelRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *panelRenderer) Refresh()                     { canvas.Refresh(r.panel) }
