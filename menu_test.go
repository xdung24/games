package main

import (
	"fmt"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

func TestGameMenuKeyboardNavigation(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	w := test.NewWindow(widget.NewLabel(""))
	defer w.Close()

	launched, launches, fallbackKeys := -1, 0, 0
	w.Canvas().SetOnTypedKey(func(*fyne.KeyEvent) { fallbackKeys++ })
	entries := make([]gameEntry, 21)
	for i := range entries {
		entries[i] = gameEntry{
			name: fmt.Sprintf("Game %d", i+1),
			play: func() {
				launched = i
				launches++
				w.SetContent(widget.NewLabel("Playing"))
			},
		}
	}
	menu := newGameMenu(entries)
	menu.Show(w)
	w.Resize(fyne.NewSize(640, 350))
	press := func(key fyne.KeyName) {
		event := &fyne.KeyEvent{Name: key}
		if focused := w.Canvas().Focused(); focused != nil {
			focused.TypedKey(event)
		} else if handler := w.Canvas().OnTypedKey(); handler != nil {
			handler(event)
		}
	}
	check := func(index int) {
		t.Helper()
		card := menu.list.Objects[index%gamesPerPage].(*gameCard)
		if w.Canvas().Focused() != card || !card.focused ||
			card.AccessibilityLabel() != fmt.Sprintf("Game %d", index+1) {
			t.Fatalf("keyboard selection must focus game %d", index+1)
		}
		if launches != 0 {
			t.Fatal("arrow navigation must not launch a game")
		}
	}
	check(0)
	press(fyne.KeyUp)
	press(fyne.KeyLeft)
	check(0)
	for i := 1; i < gamesPerPage; i++ {
		press(fyne.KeyDown)
	}
	check(9)
	card := menu.list.Objects[9]
	if menu.scroll.Offset.Y <= 0 ||
		card.Position().Y+card.Size().Height > menu.scroll.Offset.Y+menu.scroll.Size().Height+0.01 {
		t.Fatal("keyboard selection must scroll the selected card into view")
	}
	press(fyne.KeyDown)
	check(9)
	press(fyne.KeyRight)
	check(19)
	press(fyne.KeyRight)
	check(20)
	press(fyne.KeyRight)
	press(fyne.KeyDown)
	check(20)
	press(fyne.KeyLeft)
	check(10)
	press(fyne.KeyUp)
	check(10)
	w.Canvas().Focus(menu.next)
	press(fyne.KeyDown)
	check(11)
	w.Canvas().Unfocus()
	press(fyne.KeyDown)
	check(12)
	w.Canvas().Focus(menu.list.Objects[5].(*gameCard))
	press(fyne.KeyUp)
	check(14)
	press(fyne.KeyReturn)
	if launched != 14 || launches != 1 {
		t.Fatal("Enter must launch the selected game exactly once")
	}
	w.Canvas().Unfocus()
	press(fyne.KeyLeft)
	if fallbackKeys != 1 || menu.page != 1 {
		t.Fatal("menu navigation must stop handling keys while a game is open")
	}
	menu.Show(w)
	if w.Canvas().Focused() != menu.list.Objects[4].(*gameCard) {
		t.Fatal("returning Home must restore the selected game")
	}
}

func TestGameMenuKeyboardEmptyAndSingleGame(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()
	for _, count := range []int{0, 1} {
		t.Run(fmt.Sprint(count), func(t *testing.T) {
			launches := 0
			entries := make([]gameEntry, count)
			if count == 1 {
				entries[0] = gameEntry{name: "Only game", play: func() { launches++ }}
			}
			menu := newGameMenu(entries)
			w := test.NewWindow(widget.NewLabel(""))
			defer w.Close()
			menu.Show(w)
			defer menu.Hide()
			for _, key := range []fyne.KeyName{fyne.KeyUp, fyne.KeyDown, fyne.KeyLeft, fyne.KeyRight} {
				event := &fyne.KeyEvent{Name: key}
				if focused := w.Canvas().Focused(); focused != nil {
					focused.TypedKey(event)
				} else {
					w.Canvas().OnTypedKey()(event)
				}
			}
			if menu.page != 0 || menu.selected != 0 || launches != 0 {
				t.Fatal("arrow keys must not navigate past available games or pages")
			}
			if count == 1 {
				w.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
				if launches != 1 {
					t.Fatal("Enter must launch the only game")
				}
			}
		})
	}
}

func TestGameMenuPagination(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	selected := -1
	entries := make([]gameEntry, 21)
	for i := range entries {
		entries[i] = gameEntry{
			name: fmt.Sprintf("Game %d", i+1),
			play: func() { selected = i },
		}
	}
	menu := newGameMenu(entries)
	check := func(count int, first, last, status string, previousDisabled, nextDisabled bool) {
		t.Helper()
		if len(menu.list.Objects) != count {
			t.Fatalf("got %d games, want %d", len(menu.list.Objects), count)
		}
		if got := menu.list.Objects[0].(*gameCard).AccessibilityLabel(); got != first {
			t.Errorf("first game = %q, want %q", got, first)
		}
		if got := menu.list.Objects[count-1].(*gameCard).AccessibilityLabel(); got != last {
			t.Errorf("last game = %q, want %q", got, last)
		}
		if menu.status.Text != status {
			t.Errorf("page status = %q, want %q", menu.status.Text, status)
		}
		if menu.previous.Disabled() != previousDisabled || menu.next.Disabled() != nextDisabled {
			t.Error("pagination buttons have incorrect enabled state")
		}
	}

	check(10, "Game 1", "Game 10", "Page 1 of 3", true, false)
	test.Tap(menu.previous)
	check(10, "Game 1", "Game 10", "Page 1 of 3", true, false)
	test.Tap(menu.next)
	check(10, "Game 11", "Game 20", "Page 2 of 3", false, false)
	test.Tap(menu.list.Objects[0].(*gameCard))
	if selected != 10 {
		t.Errorf("selected game = %d, want 10", selected)
	}
	test.Tap(menu.next)
	check(1, "Game 21", "Game 21", "Page 3 of 3", false, true)
	test.Tap(menu.next)
	check(1, "Game 21", "Game 21", "Page 3 of 3", false, true)
	test.Tap(menu.previous)
	check(10, "Game 11", "Game 20", "Page 2 of 3", false, false)
}

func TestGameMenuSingleGame(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	played := false
	menu := newGameMenu([]gameEntry{{name: "game", play: func() { played = true }}})
	if len(menu.list.Objects) != 1 || menu.status.Text != "Page 1 of 1" ||
		!menu.previous.Disabled() || !menu.next.Disabled() {
		t.Fatal("single-game menu must show one game and disable pagination")
	}
	card := menu.list.Objects[0].(*gameCard)
	text := card.Content.(*fyne.Container)
	if _, ok := text.Objects[1].(*widget.Label); !ok {
		t.Fatal("game card must display its description without a Play button")
	}
	test.Tap(card)
	if !played {
		t.Error("tapping the card did not launch the game")
	}
}

func TestGameCardWholeAreaAndKeyboard(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	launches := 0
	card := newGameCard(gameEntry{
		name: "game", description: "Break the blocks.",
		icon: fyne.NewStaticResource("brick-breaker-Icon.png", brickBreakerIconPNG),
		play: func() { launches++ },
	})
	w := test.NewWindow(card)
	defer w.Close()
	w.Resize(fyne.NewSize(400, 200))
	for _, pos := range []fyne.Position{
		fyne.NewPos(10, 10), fyne.NewPos(200, 100), fyne.NewPos(390, 190),
	} {
		before := launches
		test.TapCanvas(w.Canvas(), pos)
		if launches != before+1 {
			t.Errorf("tap at %v did not launch the game exactly once", pos)
		}
	}

	for _, key := range []fyne.KeyName{fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter} {
		before := launches
		card.TypedKey(&fyne.KeyEvent{Name: key})
		if launches != before+1 {
			t.Errorf("key %q did not launch the game exactly once", key)
		}
	}
}

func TestGameMenuIcons(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	resource := fyne.NewStaticResource("brick-breaker-Icon.png", brickBreakerIconPNG)
	menu := newGameMenu([]gameEntry{{
		name: "Brick Breaker", description: "Break the blocks.", icon: resource,
		play: func() {},
	}})
	w := test.NewWindow(menu.content)
	defer w.Close()
	w.Resize(fyne.NewSize(640, 600))

	var homeImage *canvas.Image
	var findHomeImage func(fyne.CanvasObject)
	findHomeImage = func(object fyne.CanvasObject) {
		if image, ok := object.(*canvas.Image); ok && image.Resource == homeIcon {
			homeImage = image
		}
		if content, ok := object.(*fyne.Container); ok {
			for _, child := range content.Objects {
				findHomeImage(child)
			}
		}
	}
	findHomeImage(menu.content)
	if homeImage == nil {
		t.Fatal("Home menu must display the embedded root icon")
	}
	if homeImage.FillMode != canvas.ImageFillContain {
		t.Error("Home icon must preserve its aspect ratio")
	}
	card := menu.list.Objects[0].(*gameCard)
	content := card.Content.(*fyne.Container)
	text := content.Objects[0].(*fyne.Container)
	iconContainer := content.Objects[1].(*fyne.Container)
	icon := iconContainer.Objects[0].(*canvas.Image)
	if icon.Resource != resource || icon.FillMode != canvas.ImageFillContain {
		t.Error("game card must display its own icon with its aspect ratio preserved")
	}
	if iconContainer.Position().X+iconContainer.Size().Width > text.Position().X {
		t.Error("game icon must be to the left of the title and description")
	}
}

func TestGameCardHoverAndFocus(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newGameCard(gameEntry{name: "game", play: func() {
		t.Error("highlighting a card must not launch the game")
	}})
	r := test.WidgetRenderer(card).(*gameCardRenderer)
	check := func(highlighted bool) {
		t.Helper()
		_, _, _, fill := r.outline.FillColor.RGBA()
		_, _, _, stroke := r.outline.StrokeColor.RGBA()
		if (fill > 0) != highlighted || (stroke > 0) != highlighted {
			t.Errorf("background and border highlight = (%d, %d), want highlighted=%v", fill, stroke, highlighted)
		}
	}
	check(false)
	card.MouseIn(&desktop.MouseEvent{})
	check(true)
	card.FocusGained()
	card.MouseOut()
	check(true)
	card.FocusLost()
	check(false)
	card.FocusGained()
	check(true)
	card.MouseIn(&desktop.MouseEvent{})
	card.FocusLost()
	check(true)
	card.MouseOut()
	check(false)
}

func TestGameCardHighlightFadeReversal(t *testing.T) {
	a := test.NewApp()
	defer a.Quit()

	card := newGameCard(gameEntry{name: "game"})
	r := test.WidgetRenderer(card).(*gameCardRenderer)
	defer r.Destroy()
	card.MouseIn(&desktop.MouseEvent{})
	r.animation.Tick(0)
	_, _, _, startAlpha := r.outline.FillColor.RGBA()
	r.animation.Tick(0.5)
	_, _, _, middleAlpha := r.outline.FillColor.RGBA()
	r.animation.Tick(1)
	_, _, _, endAlpha := r.outline.FillColor.RGBA()
	if startAlpha != 0 || middleAlpha == 0 || middleAlpha >= endAlpha {
		t.Fatal("hover must fade through an intermediate background color")
	}
	r.animation.Tick(0.5)
	middleFill, middleStroke := r.outline.FillColor, r.outline.StrokeColor
	card.MouseOut()
	r.animation.Tick(0)
	if r.outline.FillColor != middleFill || r.outline.StrokeColor != middleStroke {
		t.Error("reversing a fade must start from the current background and border colors")
	}
	r.animation.Tick(1)
	_, _, _, fill := r.outline.FillColor.RGBA()
	_, _, _, stroke := r.outline.StrokeColor.RGBA()
	if fill != 0 || stroke != 0 {
		t.Error("leaving the card must fade both background and border back to transparent")
	}
}
