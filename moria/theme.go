package moria

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type gameTheme struct {
	fyne.Theme
}

func NewTheme() fyne.Theme {
	return &gameTheme{Theme: theme.DefaultTheme()}
}

func (t *gameTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == theme.ColorNameBackground {
		return colCave
	}
	return t.Theme.Color(name, variant)
}
