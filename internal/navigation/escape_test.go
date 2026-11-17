package navigation

import (
	"testing"

	"fyne.io/fyne/v2"
)

func TestEscapeRequiresSeparatePresses(t *testing.T) {
	var h EscapeHandler
	pauses, backs := 0, 0
	pause := func() { pauses++ }
	back := func() { backs++ }
	h.TypedKey(fyne.KeyEscape, pause, back)
	h.TypedKey(fyne.KeyEscape, pause, back)
	if pauses != 1 || backs != 0 {
		t.Fatal("holding Esc must pause once without returning Home")
	}
	h.KeyUp(fyne.KeyEscape)
	h.TypedKey(fyne.KeyEscape, pause, back)
	if pauses != 1 || backs != 1 {
		t.Fatal("the second Esc press must return Home")
	}
}

func TestEscapeSequenceCancellation(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(map[bool]string{false: "other-key", true: "focus-or-pointer"}[reset], func(t *testing.T) {
			var h EscapeHandler
			pauses, backs := 0, 0
			pause := func() { pauses++ }
			back := func() { backs++ }
			h.TypedKey(fyne.KeyEscape, pause, back)
			h.KeyUp(fyne.KeyEscape)
			if reset {
				h.Reset()
			} else if h.TypedKey(fyne.KeyP, pause, back) {
				t.Fatal("non-Esc input must remain available to the game")
			}
			h.TypedKey(fyne.KeyEscape, pause, back)
			if pauses != 2 || backs != 0 {
				t.Fatal("other input or losing focus must cancel the pending return Home")
			}
		})
	}
}
