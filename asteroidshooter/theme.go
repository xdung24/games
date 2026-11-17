package asteroidshooter

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// gameTheme wraps the default Fyne theme and substitutes a near-black space
// background. This matters on mobile, where the area outside the canvas's
// interactive region is painted with the theme background; without this it shows
// a default grey that clashes with the neon play surface.
type gameTheme struct {
	fyne.Theme
}

// NewTheme returns the game's space-coloured theme.
func NewTheme() fyne.Theme {
	return &gameTheme{Theme: theme.DefaultTheme()}
}

func (t *gameTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground {
		return color.NRGBA{0x03, 0x02, 0x08, 0xFF}
	}
	return t.Theme.Color(name, variant)
}
