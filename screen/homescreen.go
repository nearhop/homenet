//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
)

const status_width = 37
const status_height = 16
const status_width_offset = 262
const status_height_offset = 26
const status_value1_width = 10
const status_value1_height = 10
const status_value1_width_offset = 468
const status_value1_height_offset = 29
const status_value2_width = 61
const status_value2_height = 16
const status_value2_width_offset = 483
const status_value2_height_offset = 26
const name_width = 35
const name_height = 16
const name_width_offset = 262
const name_height_offset = 64
const name_value_width = 80
const name_value_height = 16
const name_value_width_offset = 468
const name_value_height_offset = 64
const myip_width = 82
const myip_height = 16
const myip_width_offset = 262
const myip_height_offset = 102
const myip_value_width = 120
const myip_value_height = 16
const myip_value_width_offset = 468
const myip_value_height_offset = 102
const relayip_width = 122
const relayip_height = 16
const relayip_width_offset = 262
const relayip_height_offset = 140
const relayip_value_width = 120
const relayip_value_height = 16
const relayip_value_width_offset = 468
const relayip_value_height_offset = 140
const lastupdated_width = 78
const lastupdated_height = 16
const lastupdated_width_offset = 262
const lastupdated_height_offset = 178
const lastupdated_value_width = 150
const lastupdated_value_height = 16
const lastupdated_value_width_offset = 468
const lastupdated_value_height_offset = 178
const version_width = 78
const version_height = 16
const version_width_offset = 262
const version_height_offset = 216
const version_value_width = 150
const version_value_height = 16
const version_value_width_offset = 468
const version_value_height_offset = 216
const curversion_width = 78
const curversion_height = 16
const curversion_width_offset = 262
const curversion_height_offset = 244
const curversion_value_width = 150
const curversion_value_height = 16
const curversion_value_width_offset = 468
const curversion_value_height_offset = 244
const submitlogs_bg_width = 120
const submitlogs_bg_height = 32
const submitlogs_bg_width_offset = 253
const submitlogs_bg_height_offset = 304
const submitlogs_width = 120
const submitlogs_height = 32
const submitlogs_width_offset = 254
const submitlogs_height_offset = 301
const connect_bg_width = 120
const connect_bg_height = 32
const connect_bg_width_offset = 383
const connect_bg_height_offset = 304
const connect_width = 120
const connect_height = 32
const connect_width_offset = 383
const connect_height_offset = 301

type LinkStatus struct {
	Name        string
	Version     string
	CurVersion  string
	MyVPNIP     string
	RelayHostIP string
	Status      LinkStatusType
	LastUpdated string
}

type HomeScreen struct {
	lStatus *LinkStatus
	m       *MainWindow
}

func NewLinkStatus() *LinkStatus {
	return &LinkStatus{}
}

func (h *HomeScreen) connect() {
	query := "Disconnect? (Please reset your WiFi etc and close the window if connect does n't work)"
	if h.lStatus.Status != LINK_CONNECTED {
		query = "Connect?"
	}
	dialog.ShowConfirm("Confirm", "Are you sure you want to "+query, func(confirm bool) {
		if confirm {
			if h.lStatus.Status == LINK_CONNECTED {
				CommandCallback(Disconnect, nil, 0)
			} else {
				CommandCallback(Connect, nil, 0)
			}
		}
	}, h.m.w)
}

func NewHomeScreen(name string, m *MainWindow) *HomeScreen {
	l := NewLinkStatus()

	l.Name = name
	homeScreen := &HomeScreen{
		lStatus: l,
	}
	homeScreen.m = m
	return homeScreen
}

func (h *HomeScreen) submitLogs() {
	_, err := CommandCallback(UploadLogs, nil, 0)
	if err == nil {
		h.m.ShowAlert("Logs submitted successfully")
	} else {
		h.m.ShowAlert("Error " + err.Error())
	}
}

func (h *HomeScreen) Show() fyne.CanvasObject {
	home_text_color := color.NRGBA{R: 0x01, G: 0x07, B: 0x1F, A: 255}
	names_font := fyne.TextStyle{Bold: true}

	status := newLabel("Status", home_text_color, 12, names_font)
	status.Move(fyne.Position{status_width_offset, status_height_offset})
	status.Resize(fyne.NewSize(status_width, status_height))

	var c color.NRGBA
	var connectstring string
	value2_width := status_value2_width
	if h.lStatus.Status == LINK_CONNECTED {
		c = color.NRGBA{R: 0x18, G: 0x72, B: 0x0c, A: 255}
		connectstring = "DISCONNECT"
	} else {
		c = color.NRGBA{R: 0xff, G: 0x0a, B: 0x0a, A: 255}
		value2_width += 27
		connectstring = "CONNECT"
	}
	status_value1 := canvas.NewCircle(c)
	status_value1.Move(fyne.Position{status_value1_width_offset, status_value1_height_offset})
	status_value1.Resize(fyne.NewSize(status_value1_width, status_value1_height))
	status_value1.Refresh()

	status_value2 := newLabel(LinkStatusMap[h.lStatus.Status], color.NRGBA{R: 0x00, G: 0x00, B: 0x00, A: 255}, 12, fyne.TextStyle{})
	status_value2.Move(fyne.Position{status_value2_width_offset, status_value2_height_offset})
	status_value2.Resize(fyne.NewSize(float32(value2_width), status_value2_height))

	name := newLabel("Name", home_text_color, 12, names_font)
	name.Move(fyne.Position{name_width_offset, name_height_offset})
	name.Resize(fyne.NewSize(name_width, name_height))

	name_value := newLabel(h.lStatus.Name, home_text_color, 12, fyne.TextStyle{})
	name_value.Move(fyne.Position{name_value_width_offset, name_value_height_offset})
	name_value.Resize(fyne.NewSize(name_value_width, name_value_height))

	myip := newLabel("My IP Address", home_text_color, 12, names_font)
	myip.Move(fyne.Position{myip_width_offset, myip_height_offset})
	myip.Resize(fyne.NewSize(myip_width, myip_height))

	myip_value := newLabel(h.lStatus.MyVPNIP, home_text_color, 12, fyne.TextStyle{})
	myip_value.Move(fyne.Position{myip_value_width_offset, myip_value_height_offset})
	myip_value.Resize(fyne.NewSize(myip_value_width, myip_value_height))

	relayip := newLabel("Relay Server Address", home_text_color, 12, names_font)
	relayip.Move(fyne.Position{relayip_width_offset, relayip_height_offset})
	relayip.Resize(fyne.NewSize(relayip_width, relayip_height))

	relayip_value := newLabel(h.lStatus.RelayHostIP, home_text_color, 12, fyne.TextStyle{})
	relayip_value.Move(fyne.Position{relayip_value_width_offset, relayip_value_height_offset})
	relayip_value.Resize(fyne.NewSize(relayip_value_width, relayip_value_height))

	lastupdated := newLabel("Last Updated", home_text_color, 12, names_font)
	lastupdated.Move(fyne.Position{lastupdated_width_offset, lastupdated_height_offset})
	lastupdated.Resize(fyne.NewSize(lastupdated_width, lastupdated_height))

	lastupdated_value := newLabel(h.lStatus.LastUpdated, home_text_color, 12, fyne.TextStyle{})
	lastupdated_value.Move(fyne.Position{lastupdated_value_width_offset, lastupdated_value_height_offset})
	lastupdated_value.Resize(fyne.NewSize(lastupdated_value_width, lastupdated_value_height))

	version := newLabel("Version", home_text_color, 12, names_font)
	version.Move(fyne.Position{version_width_offset, version_height_offset})
	version.Resize(fyne.NewSize(version_width, version_height))

	version_value := newLabel(h.lStatus.Version, home_text_color, 12, fyne.TextStyle{})
	version_value.Move(fyne.Position{version_value_width_offset, version_value_height_offset})
	version_value.Resize(fyne.NewSize(version_value_width, version_value_height))

	curversion := newLabel("Latest Version", home_text_color, 12, names_font)
	curversion.Move(fyne.Position{curversion_width_offset, curversion_height_offset})
	curversion.Resize(fyne.NewSize(curversion_width, curversion_height))

	curversion_value := newLabel(h.lStatus.CurVersion, home_text_color, 12, fyne.TextStyle{})
	curversion_value.Move(fyne.Position{curversion_value_width_offset, curversion_value_height_offset})
	curversion_value.Resize(fyne.NewSize(curversion_value_width, curversion_value_height))

	submitlogs_button_bg := canvas.NewImageFromResource(resourceButtonLight)
	submitlogs_button_bg.Resize(fyne.NewSize(submitlogs_bg_width, submitlogs_bg_height))
	submitlogs_button_bg.Move(fyne.Position{submitlogs_bg_width_offset, submitlogs_bg_height_offset})

	submitlogs := NewNHLabelButton("SUBMIT LOGS", h.submitLogs, h.submitLogs)

	submitlogs.Resize(fyne.NewSize(submitlogs_width, submitlogs_height))
	submitlogs.Move(fyne.Position{submitlogs_width_offset, submitlogs_height_offset})

	connect_button_bg := canvas.NewImageFromResource(resourceButtonDark)
	connect_button_bg.Resize(fyne.NewSize(connect_bg_width, connect_bg_height))
	connect_button_bg.Move(fyne.Position{connect_bg_width_offset, connect_bg_height_offset})

	connect := NewNHLabelButton(connectstring, h.connect, h.connect)
	if h.lStatus.Status == LINK_CONNECTED || h.lStatus.Status == LINK_DISCONNECTED {
		connect.Resize(fyne.NewSize(connect_width, connect_height))
		connect.Move(fyne.Position{connect_width_offset, connect_height_offset})
	} else {
		connect_button_bg.Hide()
		connect.Hide()
	}

	return container.New(NewNearhopLayout(), status, status_value1, status_value2, name, name_value, myip, myip_value, relayip, relayip_value, lastupdated, lastupdated_value, version, version_value, curversion, curversion_value, submitlogs_button_bg, submitlogs, connect_button_bg, connect)
}

func (h *HomeScreen) setHomeDetails(hd *HomeDetails) {
	h.lStatus.Status = hd.Status
	h.lStatus.Name = hd.Name
	h.lStatus.Version = hd.Version
	h.lStatus.CurVersion = hd.CurVersion
	formatted := fmt.Sprintf("%d-%02d-%02dT%02d:%02d:%02d",
		hd.LastUpdated.Year(), hd.LastUpdated.Month(), hd.LastUpdated.Day(),
		hd.LastUpdated.Hour(), hd.LastUpdated.Minute(), hd.LastUpdated.Second())
	h.lStatus.LastUpdated = formatted
	h.lStatus.MyVPNIP = hd.MyVPNIP
	h.lStatus.RelayHostIP = hd.RelayHostIP
}
