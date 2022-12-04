//go:build screen && !server
// +build screen,!server

package screen

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"
)

type KeyCallback func()
type ButtonCallback func()

type NHEntry struct {
	widget.Entry
	callback KeyCallback
}

type NHButton struct {
	widget.Button
	callback ButtonCallback
}

type NHLabelButton struct {
	widget.Label
	OnTapped func() `json:"-"`
	callback ButtonCallback
}

type NHCheck struct {
	widget.Check
	callback KeyCallback
}

func (e *NHEntry) KeyUp(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyReturn:
		e.callback()
	}
}

func (b *NHButton) KeyUp(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyReturn:
		b.callback()
	}
}
func (b *NHLabelButton) KeyUp(k *fyne.KeyEvent) {
	switch k.Name {
	case fyne.KeyReturn:
		b.callback()
	}
}

func NewNHEntry(callback KeyCallback) *NHEntry {
	entry := &NHEntry{}
	entry.callback = callback
	entry.Wrapping = fyne.TextTruncate
	entry.ExtendBaseWidget(entry)
	return entry
}

func NewNHEntryWithPlaceHolder(text string, callback KeyCallback) *NHEntry {
	entry := &NHEntry{}
	entry.callback = callback
	entry.Wrapping = fyne.TextTruncate
	entry.ExtendBaseWidget(entry)
	entry.SetPlaceHolder(text)
	return entry
}

func NewNHCheck() *NHCheck {
	entry := &NHCheck{}
	entry.ExtendBaseWidget(entry)
	return entry
}

func newLabel(value string, c color.NRGBA, size float32, style fyne.TextStyle) *canvas.Text {
	return &canvas.Text{Text: value,
		Alignment: fyne.TextAlignLeading,
		Color:     c,
		TextSize:  size,
		TextStyle: style,
	}
}

func NewNHButton(label string, tapped func(), callback ButtonCallback) *NHButton {
	//entry := widget.NewButton(label, tapped)
	entry := &NHButton{}
	entry.ExtendBaseWidget(entry)
	entry.Text = label
	entry.OnTapped = tapped
	entry.callback = callback
	return entry
}

func (nhl *NHLabelButton) Tapped(*fyne.PointEvent) {
	if nhl.OnTapped != nil {
		nhl.OnTapped()
	}
}

func NewNHLabelButton(label string, tapped func(), callback ButtonCallback) *NHLabelButton {
	entry := &NHLabelButton{}
	entry.ExtendBaseWidget(entry)
	entry.Text = label
	entry.Alignment = fyne.TextAlignCenter
	entry.OnTapped = tapped
	if callback == nil {
		entry.callback = tapped
	} else {
		entry.callback = callback
	}
	return entry
}
