package navigation

import "fyne.io/fyne/v2"

// EscapeHandler requires two separate Esc presses before leaving a game.
type EscapeHandler struct {
	pending bool
	held    bool
}

func (h *EscapeHandler) TypedKey(key fyne.KeyName, pause, onBack func()) bool {
	if key != fyne.KeyEscape {
		h.pending = false
		return false
	}
	if h.held {
		return true
	}
	h.held = true
	if h.pending {
		h.pending = false
		if onBack != nil {
			onBack()
		}
	} else {
		h.pending = true
		pause()
	}
	return true
}

func (h *EscapeHandler) KeyUp(key fyne.KeyName) {
	if key == fyne.KeyEscape {
		h.held = false
	}
}

func (h *EscapeHandler) Reset() {
	h.pending = false
	h.held = false
}
