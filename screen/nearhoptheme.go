//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type NearhopTheme struct{}

func NewNearhopTheme() *NearhopTheme {
	return &NearhopTheme{}
}

func (nt NearhopTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	if name == "placeholder" {
		//return color.Black
		return color.NRGBA{R: 0x3B, G: 0x3C, B: 0x3B, A: 255}
	}
	return theme.DarkTheme().Color(name, variant)
}

func (nt NearhopTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	if name == "radioButton" {
		return resourceRadioButton
	} else if name == "radioButtonChecked" {
		return resourceRadioButtonChecked
	}
	return theme.DarkTheme().Icon(name)
}

func (nt NearhopTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.DarkTheme().Font(style)
}

func (nt NearhopTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DarkTheme().Size(name)
}
