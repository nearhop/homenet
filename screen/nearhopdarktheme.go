//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type NearhopDarkTheme struct{}

func NewNearhopDarkTheme() *NearhopDarkTheme {
	return &NearhopDarkTheme{}
}

func (nt NearhopDarkTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DarkTheme().Color(name, variant)
}

func (nt NearhopDarkTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	if name == "radioButton" {
		return resourceRadioButton
	} else if name == "radioButtonChecked" {
		return resourceRadioButtonChecked
	}
	return theme.DarkTheme().Icon(name)
}

func (nt NearhopDarkTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (nt NearhopDarkTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}
