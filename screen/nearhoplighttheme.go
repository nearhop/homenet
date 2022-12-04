//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

type NearhopLightTheme struct{}

func NewNearhopLightTheme() *NearhopLightTheme {
	return &NearhopLightTheme{}
}

func (nt NearhopLightTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.LightTheme().Color(name, variant)
}

func (nt NearhopLightTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	if name == "radioButton" {
		return resourceRadioButton
	} else if name == "radioButtonChecked" {
		return resourceRadioButtonChecked
	}
	return theme.LightTheme().Icon(name)
}

func (nt NearhopLightTheme) Font(style fyne.TextStyle) fyne.Resource {
	return theme.LightTheme().Font(style)
}

func (nt NearhopLightTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.LightTheme().Size(name)
}
