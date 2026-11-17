package bombsweeper

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type myTheme struct{}

var _ fyne.Theme = (*myTheme)(nil)

// NewTheme returns the Bombsweeper theme.
func NewTheme() fyne.Theme {
	return &myTheme{}
}

func (m myTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	switch name {
	case theme.ColorNameBackground, theme.ColorNameMenuBackground:
		return color.NRGBA{R: 210, G: 190, B: 159, A: 255}
	case theme.ColorNameForeground:
		return color.NRGBA{A: 255}
	}
	return theme.DefaultTheme().Color(name, theme.VariantLight)
}

func (m myTheme) Font(name fyne.TextStyle) fyne.Resource {
	return theme.DefaultTheme().Font(name)
}

func (m myTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (m myTheme) Size(name fyne.ThemeSizeName) float32 {
	if name == theme.SizeNamePadding {
		return 0
	}
	return theme.DefaultTheme().Size(name)
}
