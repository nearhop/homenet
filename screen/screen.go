//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"fmt"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/slackhq/nebula/config"
)

const MAX_NUMBER_OF_MENU_ITEMS = 3
const window_width = 840
const window_height = 446
const header_height = 50
const menu_height = 35
const menu_item_width = 100
const menu_item_height = 24
const logo_width = 110
const logo_height = 18
const logo_width_offset = 34.71
const logo_height_offset = 16
const content_width = 450
const left_panel_width = 220
const menu_items_width_offset = 36
const menu_items_height_offset = 65
const space_between_menu_items = 43
const menu_icon_width = 24
const menu_icon_height = 24
const content_width_offset = 275
const content_height_offset = 25
const help_bg_width = 24
const help_bg_height = 24
const help_text_width = 80
const help_text_height = 24
const nearhop_help_url = "https://support.nearhop.com/"

type appLabels struct {
	label *canvas.Text
	icon  *canvas.Image
}

type MainWindow struct {
	a                    fyne.App
	hs                   *HomeScreen
	os                   *OnboardScreen
	ns                   *NetworkScreen
	w                    fyne.Window
	content              *fyne.Container
	curwindow            int
	hd                   *HomeDetails
	onboarded            bool
	appList              []*widget.List
	labels               []appLabels
	labelColorSelected   color.NRGBA
	labelColorUnselected color.NRGBA
	status               dialog.Dialog
	formLabelColor       color.NRGBA
}

type appInfo struct {
	name   string
	icon   fyne.Resource
	sicon  fyne.Resource
	canv   bool
	screen Screen
}

var post_onboard_apps = []appInfo{
	{"LINK                 ", resourceLink, resourceLinklight, false, nil},
	{"ADMIN                ", resourceAdmin, resourceAdminlight, false, nil},
	{"NETWORK              ", resourceNetwork, resourceNetworklight, false, nil},
}

var pre_onboard_apps = []appInfo{
	{"Onboard", nil, nil, false, nil},
}

var CommandCallback GUICommandCallback

func NewMainWindow() (*MainWindow, error) {
	m := &MainWindow{}
	m.appList = make([](*widget.List), MAX_NUMBER_OF_MENU_ITEMS)
	if m.appList == nil {
		return nil, nil
	}
	m.labels = make([](appLabels), MAX_NUMBER_OF_MENU_ITEMS)
	if m.labels == nil {
		return nil, nil
	}
	m.labelColorUnselected = color.NRGBA{R: 0x71, G: 0xB0, B: 0xCD, A: 255}
	m.labelColorSelected = color.NRGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 255}
	return m, nil
}

func (m *MainWindow) getList(apps []appInfo, curindex int, index int) *widget.List {
	m.content = container.NewMax()
	appList := widget.NewList(
		func() int {
			return len(apps)
		},
		func() fyne.CanvasObject {
			var icon fyne.Resource
			var c color.NRGBA

			if curindex == index {
				c = m.labelColorSelected
				icon = apps[0].sicon
			} else {
				c = m.labelColorUnselected
				icon = apps[0].icon
			}
			m.labels[curindex].icon = &canvas.Image{}
			m.labels[curindex].label = newLabel(apps[0].name, c, 12, fyne.TextStyle{})
			m.labels[curindex].icon.SetMinSize(fyne.NewSize(menu_icon_width, menu_icon_height))
			m.labels[curindex].icon.Resource = icon
			m.labels[curindex].icon.Refresh()
			return container.NewHBox(m.labels[curindex].icon, m.labels[curindex].label)
		},
		func(id widget.ListItemID, obj fyne.CanvasObject) {
		})
	appList.OnSelected = func(id widget.ListItemID) {
		m.curwindow = curindex
		m.reload(curindex)
		m.content.Objects = []fyne.CanvasObject{apps[id].screen.Show()}
		if m.hd != nil {
			m.SetHomeDetails(m.hd)
		}
	}
	return appList
}

func (m *MainWindow) showurl() {
	urlhelp := widget.NewLabel("Please visit https://support.nearhop.com/")
	urlhelpbox := container.NewHBox(
		urlhelp,
		widget.NewButtonWithIcon("Copy URL", theme.ContentCopyIcon(), func() {
			m.w.Clipboard().SetContent("https://support.nearhop.com/")
		}),
	)
	m.ShowAlertWithCanvasObject(urlhelpbox)
}

func (m *MainWindow) reload(index int) {
	if m.onboarded {
		m.appList[0] = m.getList(post_onboard_apps[0:1], 0, index)
		m.appList[1] = m.getList(post_onboard_apps[1:2], 1, index)
		m.appList[2] = m.getList(post_onboard_apps[2:3], 2, index)
		// Put the header first
		// A black strip with logo overlapped on it on the left
		header_bg := canvas.NewRectangle(color.NRGBA{R: 0xA6, G: 0xDF, B: 0xF0, A: 255})
		header_bg.Resize(fyne.NewSize(left_panel_width, header_height))
		logo := canvas.NewImageFromResource(resourceLogobig)
		logo.Resize(fyne.NewSize(logo_width, logo_height))
		logo.Move(fyne.Position{logo_width_offset, logo_height_offset})

		var help_height_offset float32
		help_height_offset = 0

		for i := 0; i < len(post_onboard_apps); i++ {
			m.appList[i].Move(fyne.Position{float32(menu_items_width_offset), float32(menu_items_height_offset) + float32(i*space_between_menu_items)})
			m.appList[i].Resize(fyne.NewSize(menu_item_width, menu_item_height))
			help_height_offset = float32(menu_items_height_offset) + float32(i*space_between_menu_items)
		}

		help_height_offset += float32(space_between_menu_items)
		help_bg := canvas.NewImageFromResource(resourceSupport)
		help_bg.Resize(fyne.NewSize(help_bg_width, help_bg_height))
		help_bg.Move(fyne.Position{float32(menu_items_width_offset), help_height_offset + 5})

		hlink := NewNHLabelButton("   HELP", m.showurl, m.showurl)
		hlink.Resize(fyne.NewSize(help_text_width, help_text_height))
		hlink.Move(fyne.Position{float32(menu_items_width_offset), help_height_offset})

		m.content.Objects = []fyne.CanvasObject{post_onboard_apps[index].screen.Show()}

		right_panel_bg := canvas.NewRectangle(color.NRGBA{R: 96, G: 161, B: 193, A: 255})
		right_panel_bg.Resize(fyne.NewSize(window_width-left_panel_width, window_height))
		right_panel_bg.Move(fyne.Position{float32(left_panel_width), 0})
		right_panel := container.New(NewNearhopLayout(), right_panel_bg, m.content)

		left_bg_height := float32(window_height)
		if right_panel.MinSize().Height > left_bg_height {
			left_bg_height = right_panel.MinSize().Height
		}
		left_panel_bg := canvas.NewRectangle(color.NRGBA{R: 0x3C, G: 0x7A, B: 0x99, A: 255})
		left_panel_bg.Resize(fyne.NewSize(left_panel_width, left_bg_height))
		left_panel := container.New(NewNearhopLayout(), left_panel_bg, header_bg, logo, m.appList[0], m.appList[1], m.appList[2], help_bg)
		if hlink != nil {
			left_panel = container.New(NewNearhopLayout(), left_panel_bg, header_bg, logo, m.appList[0], m.appList[1], m.appList[2], help_bg, hlink)
		}

		split := container.NewVScroll(container.New(NewNearhopLayout(), left_panel, right_panel))

		m.w.SetPadded(false)
		m.w.SetContent(split)
		m.w.Resize(fyne.NewSize(window_width, window_height))
		m.w.SetFixedSize(true)
	} else {
		m.getList(pre_onboard_apps[0:1], 0, index)
		m.content.Objects = []fyne.CanvasObject{pre_onboard_apps[0].screen.Show()}
		m.w.SetContent(m.content)
		m.w.Resize(fyne.NewSize(onboard_window_width, onboard_window_height))
		m.w.SetFixedSize(true)
	}
}

func (m *MainWindow) StartMainWindow(onboarded bool, c *config.C, cert string, status_err string, callback GUICommandCallback) error {
	m.a = app.New()
	m.a.SetIcon(resourceLogo)
	if onboarded {
		m.a.Settings().SetTheme(NewNearhopTheme())
	} else {
		m.a.Settings().SetTheme(NewNearhopLightTheme())
	}

	m.w = m.a.NewWindow("Nearhop")
	m.curwindow = 0
	m.w.Canvas().SetOnTypedKey(func(k *fyne.KeyEvent) {
	})
	m.w.SetIcon(resourceLogo)
	m.w.SetPadded(false)
	m.w.Resize(fyne.NewSize(window_width, window_height))
	m.w.SetFixedSize(true)

	m.w.SetOnClosed(func() {
		callback(StopMain, nil, 0)
	})
	if onboarded {
		m.formLabelColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	} else {
		m.formLabelColor = color.NRGBA{R: 01, G: 07, B: 31, A: 255}
	}
	CommandCallback = callback

	// Home screen
	hs1 := NewHomeScreen("", m)
	if hs1 == nil {
		return fmt.Errorf("Error while initing HomeScreen")
	}
	post_onboard_apps[0].screen = hs1

	// Admin screen
	as1 := NewAdminScreen(m, c)
	if as1 == nil {
		return fmt.Errorf("Error while initing AdminScreen")
	}
	post_onboard_apps[1].screen = as1

	// Network screen
	ns1 := NewNetworkScreen()
	if ns1 == nil {
		return fmt.Errorf("Error while initing AdminScreen")
	}
	post_onboard_apps[2].screen = ns1

	// Onboarding screen
	os1 := NewOnboardScreen(m, cert)
	if os1 == nil {
		return fmt.Errorf("Error while initing OnboardScreen")
	}
	pre_onboard_apps[0].screen = os1

	m.hs = hs1
	m.os = os1
	m.ns = ns1
	m.onboarded = onboarded

	if onboarded {
		m.reload(0)
	} else {
		m.reload(0)
	}
	m.w.ShowAndRun()
	m.hd = nil
	m.status = nil

	return nil
}

func (m *MainWindow) SetHomeDetails(hd *HomeDetails) {
	if m.hs == nil {
		// Application not inited
		return
	}
	m.hd = hd
	m.hs.setHomeDetails(hd)
	if m.curwindow == 2 {
		objs := []fyne.CanvasObject{m.ns.setNetworkDetails(hd)}
		if objs != nil {
			m.content.Objects = objs
		}
	}
	if m.onboarded && (m.curwindow == 0 || m.curwindow == 2) {
		m.reload(m.curwindow)
	}
}

func (m *MainWindow) Onboarded(ip string) {
	m.onboarded = true
	m.a.Settings().SetTheme(NewNearhopTheme())
	m.formLabelColor = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	m.reload(0)
}

func (m *MainWindow) ReloadAdmin() {
	if m.onboarded {
		m.reload(1)
	}
}

func (m *MainWindow) ShowAlert(s string) {
	status := widget.NewLabel(s)
	statusbox := container.NewHBox(
		status,
		widget.NewButtonWithIcon("Copy", theme.ContentCopyIcon(), func() {
			m.w.Clipboard().SetContent(s)
		}),
	)
	if m.status != nil {
		m.status.Hide()
	}
	m.status = dialog.NewCustom("Alert", "OK", statusbox, m.w)
	m.status.Show()
}

func (m *MainWindow) ShowAlertWithCanvasObject(status_object fyne.CanvasObject) {
	if m.status != nil {
		m.status.Hide()
	}
	m.status = dialog.NewCustom("Alert", "OK", status_object, m.w)
	m.status.Show()
}
