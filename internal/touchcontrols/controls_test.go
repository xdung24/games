package touchcontrols

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/driver/mobile"
	"fyne.io/fyne/v2/test"
)

func TestButtonKeepsIndependentTouchAndMouseHolds(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	downs, ups := 0, 0
	b := NewButton("Left", LeftIcon, func() { downs++ }, func() { ups++ })
	b.TouchDown(&mobile.TouchEvent{ID: 1})
	b.TouchDown(&mobile.TouchEvent{ID: 1})
	b.TouchDown(&mobile.TouchEvent{ID: 2})
	b.TouchCancel(&mobile.TouchEvent{ID: 1})
	if downs != 1 || ups != 0 || len(b.held) != 1 {
		t.Fatal("one finger must not cancel another or duplicate its action")
	}
	b.TouchUp(&mobile.TouchEvent{ID: 2})
	b.TouchUp(&mobile.TouchEvent{ID: 2})
	if ups != 1 {
		t.Fatal("the last touch must release exactly once")
	}
	b.MouseDown(&desktop.MouseEvent{Button: desktop.MouseButtonPrimary})
	b.TouchDown(&mobile.TouchEvent{ID: 3})
	b.MouseOut()
	if downs != 2 || ups != 1 {
		t.Fatal("mouse exit must not cancel a remaining touch")
	}
	b.Reset()
	b.TouchUp(&mobile.TouchEvent{ID: 3})
	if ups != 2 || len(b.held) != 0 {
		t.Fatal("reset must release once and discard stale touches")
	}
}

func TestPanelLayoutReservesPlayfieldAndSeparatesHands(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	button := func(name string) *Button { return NewButton(name, LeftIcon, nil, nil) }
	content := canvas.NewRectangle(color.Black)
	p := NewPanel(content, []*Button{button("Left"), button("Right")},
		nil, []*Button{button("Thrust"), button("Fire")},
		[]*Button{button("Pause"), button("Mute")})
	w := test.NewWindow(p)
	defer w.Close()
	for _, size := range []fyne.Size{fyne.NewSize(320, 568), fyne.NewSize(430, 740), fyne.NewSize(932, 430)} {
		w.Resize(size)
		p.Resize(size)
		if content.Size().Height != size.Height-Height {
			t.Fatalf("controls do not reserve playfield space at %v", size)
		}

		for _, group := range [][]*Button{p.Left, p.Center, p.Right, p.System} {
			for _, b := range group {
				pos, s := b.Position(), b.Size()
				if s.Width < 44 || s.Height < 44 || pos.X < 0 || pos.Y < 0 ||
					pos.X+s.Width > size.Width || pos.Y+s.Height > size.Height {
					t.Fatalf("%s has an unreachable touch target at %v", b.Name, size)
				}
				if p.hit(pos.X+s.Width/2, pos.Y+s.Height/2, size.Width, size.Height) != b {
					t.Fatalf("pointer translation misses %s at %v", b.Name, size)
				}
			}
		}
		if p.Left[1].Position().X+p.Left[1].Size().Width >= p.Right[0].Position().X {
			t.Fatalf("gameplay groups overlap at %v", size)
		}
	}
	if p.hit(10, 200, 932, 430) != nil {
		t.Fatal("ordinary playfield input must pass through to the game")
	}
}

func TestOverlayLayoutUpdatesTouchTargets(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	content := canvas.NewRectangle(color.Black)
	button := NewButton("Left", LeftIcon, nil, nil)
	p := NewPanel(content, []*Button{button}, nil, nil, nil)
	p.Overlay = true
	w := test.NewWindow(p)
	defer w.Close()
	renderer := test.WidgetRenderer(p).(*panelRenderer)
	for _, size := range []fyne.Size{fyne.NewSize(320, 568), fyne.NewSize(430, 740), fyne.NewSize(932, 430)} {
		p.Resize(size)
		pos, s := button.Position(), button.Size()
		if renderer.footer.Visible() || content.Size() != size || pos.Y+s.Height >= size.Height ||
			p.hit(pos.X+s.Width/2, pos.Y+s.Height/2, size.Width, size.Height) != button {
			t.Fatalf("overlay must preserve the full playfield and route touches to its visible button at %v", size)
		}
		if p.hit(size.Width/2, size.Height/2, size.Width, size.Height) != nil {
			t.Fatal("uncovered playfield touches must pass through")
		}
	}
}
