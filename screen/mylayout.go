// Package layout defines the various layouts available to Fyne apps

// Venkat: This is an example that I generally use for debugging

package screen

import (
	"fyne.io/fyne/v2"
)

// Declare conformity with Layout interface
var _ fyne.Layout = (*myLayout)(nil)

type myLayout struct {
}

// NewMaxLayout creates a new MaxLayout instance
func NewMyLayout() fyne.Layout {
	return &myLayout{}
}

// Layout is called to pack all child objects into a specified size.
// For MaxLayout this sets all children to the full size passed.
func (g *myLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
}

func (g *myLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	var minx, maxx, miny, maxy float32

	minx = 100000 // Some big number
	maxx = 0
	miny = 100000
	maxy = 0

	count := 0
	for _, child := range objects {
		if !child.Visible() {
			continue
		}
		width := child.MinSize().Width
		height := child.MinSize().Height
		if (width + child.Position().X) > maxx {
			maxx = width + child.Position().X
		}
		if child.Position().X < minx {
			minx = child.Position().X
		}
		if (height + child.Position().Y) > maxy {
			maxy = height + child.Position().Y
		}
		if child.Position().Y < miny {
			miny = child.Position().Y
		}
		count++
	}
	if count == 0 {
		minx = 0
		maxx = 0
		miny = 0
		maxy = 0
	}
	return fyne.NewSize(maxx-minx, maxy-miny)
}
