package touchcontrols

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/widget"
)

var (
	background = color.NRGBA{0x18, 0x2B, 0x42, 0xFF}
	pressed    = color.NRGBA{0x34, 0x60, 0x79, 0xFF}
	outline    = color.NRGBA{0x5E, 0xF7, 0xFF, 0xFF}
)

// Button invokes an action once on the first press and releases after the last
// finger leaves. It does not take keyboard focus away from the game.
type Button struct {
	widget.BaseWidget
	Name        string
	Translucent bool
	icon        fyne.Resource
	down        func()
	up          func()
	held        map[int]bool
}

func NewButton(name string, icon fyne.Resource, down, up func()) *Button {
	b := &Button{Name: name, icon: icon, down: down, up: up, held: make(map[int]bool)}
	b.ExtendBaseWidget(b)
	return b
}

func (b *Button) press(id int) {
	if b.held[id] {
		return
	}
	b.held[id] = true
	if len(b.held) == 1 {
		b.Refresh()
		if b.down != nil {
			b.down()
		}
	}
}

func (b *Button) release(id int) {
	if !b.held[id] {
		return
	}
	delete(b.held, id)
	if len(b.held) == 0 {
		b.Refresh()
		if b.up != nil {
			b.up()
		}
	}
}

func (b *Button) Reset() {
	if len(b.held) == 0 {
		return
	}
	clear(b.held)
	b.Refresh()
	if b.up != nil {
		b.up()
	}
}

func (b *Button) TouchDown(e *mobile.TouchEvent)   { b.press(e.ID) }
func (b *Button) TouchUp(e *mobile.TouchEvent)     { b.release(e.ID) }
func (b *Button) TouchCancel(e *mobile.TouchEvent) { b.release(e.ID) }
func (b *Button) MouseDown(e *desktop.MouseEvent) {
	if e.Button == desktop.MouseButtonPrimary {
		b.press(-1)
	}
}
func (b *Button) MouseUp(e *desktop.MouseEvent) {
	if e.Button == desktop.MouseButtonPrimary {
		b.release(-1)
	}
}
func (b *Button) MouseIn(*desktop.MouseEvent)    {}
func (b *Button) MouseMoved(*desktop.MouseEvent) {}
func (b *Button) MouseOut()                      { b.release(-1) }

func (b *Button) fillColor() color.NRGBA {
	fill := background
	if len(b.held) > 0 {
		fill = pressed
	}
	if b.Translucent {
		fill.A = 112
		if len(b.held) > 0 {
			fill.A = 176
		}
	}
	return fill
}

func (b *Button) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(b.fillColor())
	bg.CornerRadius, bg.StrokeWidth, bg.StrokeColor = 8, 1, outline
	icon := canvas.NewImageFromResource(b.icon)
	icon.FillMode = canvas.ImageFillContain
	return &buttonRenderer{button: b, bg: bg, icon: icon, objects: []fyne.CanvasObject{bg, icon}}
}

type buttonRenderer struct {
	button  *Button
	bg      *canvas.Rectangle
	icon    *canvas.Image
	objects []fyne.CanvasObject
}

func (r *buttonRenderer) Layout(size fyne.Size) {
	r.bg.Resize(size)
	r.icon.Resize(fyne.NewSize(28, 28))
	r.icon.Move(fyne.NewPos((size.Width-28)/2, (size.Height-28)/2))
}
func (r *buttonRenderer) MinSize() fyne.Size           { return fyne.NewSize(44, 44) }
func (r *buttonRenderer) Destroy()                     {}
func (r *buttonRenderer) Objects() []fyne.CanvasObject { return r.objects }
func (r *buttonRenderer) Refresh() {
	r.bg.FillColor = r.button.fillColor()
	r.bg.Refresh()
}

func icon(name, paths string) fyne.Resource {
	return fyne.NewStaticResource(name+".svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="#e6f6ff" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">`+paths+`</svg>`))
}

var (
	LeftIcon   = icon("left", `<path d="M19 12H5m7-7-7 7 7 7"/>`)
	RightIcon  = icon("right", `<path d="M5 12h14m-7-7 7 7-7 7"/>`)
	ThrustIcon = icon("thrust", `<path d="M12 3c1 4-2 5-2 8 0 1 1 2 2 2 2 0 3-2 3-4 3 3 4 5 3 8-1 3-4 5-7 4-4-1-6-5-5-8 1-3 3-5 6-10Z"/>`)
	FireIcon   = icon("fire", `<circle cx="12" cy="12" r="6"/><path d="M12 2v4m0 12v4M2 12h4m12 0h4"/><circle cx="12" cy="12" r="1"/>`)
	PauseIcon  = icon("pause", `<path d="M9 5v14M15 5v14"/>`)
	MuteIcon   = icon("mute", `<path d="m11 5-5 4H3v6h3l5 4V5ZM16 9l5 6m0-6-5 6"/>`)
)
