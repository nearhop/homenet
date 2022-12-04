//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"image/color"
	"sort"

	nh_util "nh_util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

const items_per_row = 12
const network_item_width = 100
const network_item_height = 20
const network_items_start_offset = left_panel_width + 30

type NetworkScreen struct {
	form *fyne.Container
}

func NewNetworkForm() *fyne.Container {
	form := &fyne.Container{Layout: NewNearhopLayout()}

	//objects := make([]fyne.CanvasObject, 16)
	//form.Objects = append(form.Objects, objects...)
	return form
}

func NewNetworkScreen() *NetworkScreen {
	f := NewNetworkForm()

	networkScreen := &NetworkScreen{
		form: f,
	}
	return networkScreen
}

func (n *NetworkScreen) Show() fyne.CanvasObject {
	image := &canvas.Image{FillMode: canvas.ImageFillOriginal}
	return container.NewBorder(n.form, nil, nil, nil, image)
}

func (n *NetworkScreen) setNetworkDetails(hd *HomeDetails) fyne.CanvasObject {
	network_text_color := color.NRGBA{R: 0x01, G: 0x07, B: 0x1F, A: 255}
	names_font := fyne.TextStyle{Bold: true}
	objects := make([]fyne.CanvasObject, items_per_row*len(hd.Hosts)+6)
	if objects == nil {
		return nil
	}
	objects[0] = newLabel("Name", network_text_color, 12, names_font)
	objects[1] = newLabel("IPAddress", network_text_color, 12, names_font)
	objects[2] = newLabel("Connected?", network_text_color, 12, names_font)
	objects[3] = newLabel("Direct?", network_text_color, 12, names_font)
	objects[4] = newLabel("Download", network_text_color, 12, names_font)
	objects[5] = newLabel("Upload", network_text_color, 12, names_font)
	i := 6

	var height_offset float32
	height_offset = 0

	for j := 0; j < 6; j++ {
		objects[j].Resize(fyne.NewSize(network_item_width, network_item_height))
		objects[j].Move(fyne.Position{network_items_start_offset + float32(j*network_item_width), float32(height_offset)})
	}
	height_offset += network_item_height
	height_offset += theme.Padding()

	n.form = NewNetworkForm()

	hosts := make([]*NetworkEntry, len(hd.Hosts))
	index := 0
	for _, ne := range hd.Hosts {
		hosts[index] = ne
		index++
	}

	sort.SliceStable(hosts, func(i, j int) bool {
		return hosts[i].VpnIp < hosts[j].VpnIp
	})
	for _, ne := range hosts {
		var direct string
		var connected string

		if ne.Relay == 1 {
			direct = "No"
		} else {
			direct = "Yes"
		}
		if ne.Connected {
			connected = "Yes"
		} else {
			connected = "Not yet"
			direct = "--"
		}

		in_bytes := nh_util.NH_convert_into_xB(ne.In_bytes)
		out_bytes := nh_util.NH_convert_into_xB(ne.Out_bytes)
		in_diff_bytes := nh_util.NH_convert_into_xbps(ne.In_diff_bytes, 5)
		out_diff_bytes := nh_util.NH_convert_into_xbps(ne.Out_diff_bytes, 5)

		if len(ne.Name) > 8 {
			objects[i] = newLabel(ne.Name[0:8], network_text_color, 12, fyne.TextStyle{})
		} else {
			objects[i] = newLabel(ne.Name, network_text_color, 12, fyne.TextStyle{})
		}
		objects[i+1] = newLabel((nh_util.Int2ip(ne.VpnIp)).String(), network_text_color, 12, fyne.TextStyle{})
		objects[i+2] = newLabel(connected, network_text_color, 12, fyne.TextStyle{})
		objects[i+3] = newLabel(direct, network_text_color, 12, fyne.TextStyle{})
		objects[i+4] = newLabel(in_diff_bytes, network_text_color, 12, fyne.TextStyle{})
		objects[i+5] = newLabel(out_diff_bytes, network_text_color, 12, fyne.TextStyle{})
		objects[i+6] = newLabel("", network_text_color, 12, fyne.TextStyle{})
		objects[i+7] = newLabel("", network_text_color, 12, fyne.TextStyle{})
		objects[i+8] = newLabel("", network_text_color, 12, fyne.TextStyle{})
		objects[i+9] = newLabel("", network_text_color, 12, fyne.TextStyle{})
		objects[i+10] = newLabel("("+in_bytes+")", network_text_color, 12, fyne.TextStyle{})
		objects[i+11] = newLabel("("+out_bytes+")", network_text_color, 12, fyne.TextStyle{})
		for j := 0; j < items_per_row; j++ {
			objects[i+j].Resize(fyne.NewSize(network_item_width, network_item_height))
			objects[i+j].Move(fyne.Position{network_items_start_offset + float32((j%(items_per_row/2))*network_item_width), float32(height_offset)})
			if j == (items_per_row / 2) {
				height_offset += network_item_height
			}
		}
		i = i + items_per_row
		height_offset += network_item_height
	}
	n.form.Objects = append(n.form.Objects, objects...)
	n.form.Resize(fyne.NewSize(window_width-network_items_start_offset, window_height))
	n.form.Move(fyne.Position{float32(network_items_start_offset), 0})

	right_panel_bg := canvas.NewRectangle(color.NRGBA{R: 96, G: 161, B: 193, A: 255})
	right_panel_bg.Resize(fyne.NewSize(window_width-left_panel_width, window_height))
	right_panel_bg.Move(fyne.Position{float32(left_panel_width), 0})
	right_panel_bg.SetMinSize(fyne.NewSize(window_width-left_panel_width, window_height))
	return container.New(NewNearhopLayout(), n.form)
}
