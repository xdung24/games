package main

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

type gameCard struct {
	widget.Card
	name       string
	play       func()
	focused    bool
	hovered    bool
	onFocus    func()
	onNavigate func(fyne.KeyName) bool
}

func newGameCard(entry gameEntry) *gameCard {
	description := widget.NewLabel(entry.description)
	description.Wrapping = fyne.TextWrapWord
	c := &gameCard{name: entry.name, play: entry.play}
	title := widget.NewLabelWithStyle(entry.name, fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	text := container.NewVBox(title, description)
	c.Content = text
	if entry.icon != nil {
		icon := canvas.NewImageFromResource(entry.icon)
		icon.FillMode = canvas.ImageFillContain
		icon.SetMinSize(fyne.NewSize(64, 64))
		c.Content = container.NewBorder(nil, nil, container.NewCenter(icon), nil, text)
	}
	c.ExtendBaseWidget(c)
	return c
}

func (c *gameCard) Tapped(*fyne.PointEvent) { c.play() }
func (c *gameCard) Cursor() desktop.Cursor  { return desktop.PointerCursor }
func (c *gameCard) TypedRune(rune)          {}

func (c *gameCard) MouseIn(*desktop.MouseEvent) {
	c.hovered = true
	c.Refresh()
}

func (c *gameCard) MouseMoved(*desktop.MouseEvent) {}

func (c *gameCard) MouseOut() {
	c.hovered = false
	c.Refresh()
}

func (c *gameCard) TypedKey(e *fyne.KeyEvent) {
	if c.onNavigate != nil && c.onNavigate(e.Name) {
		return
	}
	if e.Name == fyne.KeySpace || e.Name == fyne.KeyReturn || e.Name == fyne.KeyEnter {
		c.play()
	}
}

func (c *gameCard) FocusGained() {
	c.focused = true
	c.Refresh()
	if c.onFocus != nil {
		c.onFocus()
	}
}

func (c *gameCard) FocusLost() {
	c.focused = false
	c.Refresh()
}

func (c *gameCard) AccessibilityLabel() string { return c.name }
func (c *gameCard) AccessibilityRole() fyne.AccessibleRole {
	return fyne.AccessibleRoleButton
}

func (c *gameCard) CreateRenderer() fyne.WidgetRenderer {
	outline := canvas.NewRectangle(color.Transparent)
	outline.StrokeWidth = 2
	r := &gameCardRenderer{WidgetRenderer: c.Card.CreateRenderer(), card: c, outline: outline}
	r.Refresh()
	return r
}

type gameCardRenderer struct {
	fyne.WidgetRenderer
	card      *gameCard
	outline   *canvas.Rectangle
	animation *fyne.Animation
	amount    float32
	target    float32
}

func (r *gameCardRenderer) Objects() []fyne.CanvasObject {
	objects := r.WidgetRenderer.Objects()
	layered := make([]fyne.CanvasObject, 0, len(objects)+1)
	layered = append(layered, objects[0], r.outline)
	return append(layered, objects[1:]...)
}

func (r *gameCardRenderer) Destroy() {
	if r.animation != nil {
		r.animation.Stop()
	}
	r.WidgetRenderer.Destroy()
}

func (r *gameCardRenderer) Layout(size fyne.Size) {
	r.WidgetRenderer.Layout(size)
	r.outline.Move(fyne.NewPos(2, 2))
	r.outline.Resize(size.Subtract(fyne.NewSize(4, 4)))
}

func (r *gameCardRenderer) Refresh() {
	r.WidgetRenderer.Refresh()
	target := float32(0)
	if r.card.focused || r.card.hovered {
		target = 1
	}
	if target != r.target {
		if r.animation != nil {
			r.animation.Stop()
		}
		r.target = target
		start := r.amount
		r.animation = fyne.NewAnimation(canvas.DurationShort, func(progress float32) {
			r.amount = start + (target-start)*progress
			r.refreshHighlight()
		})
		if fyne.CurrentApp().Settings().ShowAnimations() {
			r.animation.Start()
		} else {
			r.animation.Tick(1)
		}
	}
	r.refreshHighlight()
}

func (r *gameCardRenderer) refreshHighlight() {
	variant := fyne.CurrentApp().Settings().ThemeVariant()
	fill := color.NRGBAModel.Convert(r.card.Theme().Color(theme.ColorNameHover, variant)).(color.NRGBA)
	stroke := color.NRGBAModel.Convert(r.card.Theme().Color(theme.ColorNameFocus, variant)).(color.NRGBA)
	fill.A = uint8(float32(fill.A) * r.amount)
	stroke.A = uint8(float32(stroke.A) * r.amount)
	r.outline.FillColor = fill
	r.outline.StrokeColor = stroke
	r.outline.CornerRadius = r.card.Theme().Size(theme.SizeNameCardRadius)
	r.outline.Refresh()
}
