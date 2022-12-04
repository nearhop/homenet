//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"fyne.io/fyne/v2"
)

type Screen interface {
	Show() fyne.CanvasObject
}
