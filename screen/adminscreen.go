//go:build screen && !server && !router
// +build screen,!server,!router

package screen

import (
	"encoding/json"
	"fmt"
	"image/color"
	"net"
	"strconv"
	"strings"

	nh_util "nh_util"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/slackhq/nebula"
	"github.com/slackhq/nebula/config"
	"github.com/slackhq/nebula/overlay"
)

const MIN_MTU = 576
const MAX_MTU = 1500
const number_of_items_in_subnet = 7
const describe_width_offset = 275
const describe_height_offset = 25
const describe_width = 136
const describe_height = 24
const mode_selector_width_offset = 275
const mode_selector_height_offset = 71
const mode_selector_width = 136
const mode_selector_height = 350
const reset_bg_width = 100
const reset_bg_height = 36
const reset_bg_width_offset = 275
const reset_bg_height_offset = 145
const save_bg_width = 100
const save_bg_height = 36
const save_bg_width_offset = 386
const save_bg_height_offset = 145
const reset_button_width = 100
const reset_button_height = 36
const reset_button_width_offset = 275
const reset_button_height_offset = 151
const save_button_width = 100
const save_button_height = 36
const save_button_width_offset = 386
const save_button_height_offset = 151
const vertical_space_between_items = 10
const form_width = 260
const form_width_offset = 275
const form_height_offset = 125
const clients_bg_width = 100
const clients_bg_height = 36
const clients_bg_width_offset = 275
const clients_bg_height_offset = 145
const clients_button_width = 87
const clients_button_height = 29
const clients_button_width_offset = 284
const clients_button_height_offset = 151
const newroute_icon_width = 100
const newroute_icon_height = 17
const newroute_icon_width_offset = 272
const newroute_text_width = 100
const newroute_text_height = 17
const newroute_text_width_offset = 272
const route_name_width_offset = 285
const route_name_width = 225
const route_name_height = 41
const route_subnet_width_offset = 525
const route_subnet_width = 225
const route_subnet_height = 41
const route_via_width_offset = 285
const route_via_width = 225
const route_via_height = 41
const route_mtu_width_offset = 525
const route_mtu_width = 225
const route_mtu_height = 41
const route_deletecheck_width_offset = 776
const route_deletecheck_width = 25
const route_deletecheck_height = 25
const route_delete_bg_width_offset = 776
const route_delete_bg_width = 25
const route_delete_bg_height = 25
const outline_width = 550
const outline_width_offset = 275
const clientsroutes_bg_width = 270
const clientsroutes_bg_height = 36
const clientsroutes_bg_width_offset = 500
const clientsroutes_bg_height_offset = 145
const clientsroutes_button_width = 270
const clientsroutes_button_height = 22
const clientsroutes_button_width_offset = 500
const clientsroutes_button_height_offset = 145

type SubnetEntry struct {
	name              *canvas.Text
	subnet            *NHEntry
	rname             *NHEntry
	via               *NHEntry
	mtu               *NHEntry
	deleteCheck       *NHLabelButton
	markedForDeletion bool
}

type AdminScreen struct {
	form                 *fyne.Container
	subnetform           *fyne.Container
	fullvpnform          *fyne.Container
	dnsserverform        *fyne.Container
	subnets              [MAX_SUBNETS]SubnetEntry
	reset                *NHLabelButton
	save                 *NHLabelButton
	add                  *NHLabelButton
	clientsRoutes        *NHLabelButton
	m                    *MainWindow
	num_of_routes        int
	radioGroup           *widget.RadioGroup
	curvpnmode           string
	config               *config.C
	newentry             bool
	wificlientsroutes    bool
	servers              [MAX_DNS_SERVERS]*NHEntry
	wifirouterip         string
	formLabelColor       color.NRGBA
	subnet_height_offset float32
	status               dialog.Dialog
}

func (a *AdminScreen) resetSubmit(w fyne.Window) {
	dialog.ShowConfirm("Confirm", "Saying yes will erase everything and close this application. You will need to onboard again.", func(confirm bool) {
		if confirm {
			CommandCallback(ResetConfig, nil, 0)
			w.Close()
		}
	}, w)
}

func (a *AdminScreen) ValidateVPNIP(config *config.C, ipstr string) error {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		fmt.Errorf("Not a valid ip address")
	}
	if nebula.ValidateVPNIP(config, ip) {
		return nil
	} else {
		return fmt.Errorf("It should be the VPN IP Address of one of your VPN devices. Refer mobile app to get the VPN IP.")
	}
}

func (a *AdminScreen) saveConfig() {
	var config Config
	entries_len := a.num_of_routes
	if a.newentry {
		entries_len++
	}
	deleted_entries := 0
	if a.curvpnmode == CustomVPN {
		for i := 0; i < entries_len; i++ {
			if a.subnets[i].markedForDeletion {
				deleted_entries++
			}
		}
	} else if a.curvpnmode == FullVPN {
		entries_len = 1
	}
	if deleted_entries > entries_len {
		a.m.ShowAlert("More entries marked for deletion than the entries length. How is it possible")
		return
	}
	entries_len -= deleted_entries

	routes := make([]RouteEntry, entries_len)
	if routes == nil {
		// Handle the error and show some proper Error Message
		a.m.ShowAlert("Nearhop Admin Error while saving. No memory")
		return
	}

	if a.curvpnmode == FullVPN {
		routes[0].Via = a.subnets[0].via.Text
		err := a.ValidateVPNIP(a.config, routes[0].Via)
		if err != nil {
			a.m.ShowAlert("Not a valid Gateway IP Address " + routes[0].Via + " " + err.Error())
			return
		}
	} else if a.curvpnmode == CustomVPN {
		index := 0
		for i := 0; i < entries_len+deleted_entries; i++ {
			if a.subnets[i].markedForDeletion {
				continue
			}
			if a.subnets[i].subnet == nil || a.subnets[i].mtu == nil {
				a.m.ShowAlert("Please click on New Route to add again")
				return
			}
			routes[index].Subnet = a.subnets[i].subnet.Text
			if !nh_util.NH_is_proper_subnet(routes[index].Subnet) {
				a.m.ShowAlert("Not a valid subnet. Example subnet value is 192.168.100.0/24. You entered " + routes[index].Subnet)
				return
			}

			routes[index].Via = a.subnets[i].via.Text
			err := a.ValidateVPNIP(a.config, routes[index].Via)
			if err != nil {
				a.m.ShowAlert("Not a valid Via. " + err.Error())
				return
			}
			routes[index].Mtu = a.subnets[i].mtu.Text
			routes[index].Name = a.subnets[i].rname.Text

			if !nh_util.NH_is_proper_Integer(routes[index].Mtu, MIN_MTU, MAX_MTU) {
				a.m.ShowAlert("Not a valid mtu  " + routes[index].Mtu + ". Valid MTU Range is " + strconv.Itoa(MIN_MTU) + " " + strconv.Itoa(MAX_MTU))
				return
			}
			index++
		}
	}

	var dnsservers string
	dnsservers = ""
	if a.curvpnmode == FullVPN || a.curvpnmode == CustomVPN {
		dnsserver1 := a.servers[0].Text
		dnsserver1 = strings.Trim(dnsserver1, " ")
		if len(dnsserver1) > 0 {
			if nh_util.NH_is_proper_ip(dnsserver1) {
				dnsservers = dnsserver1
			} else {
				a.m.ShowAlert("Invalid DNS Server address %v " + dnsserver1)
				return
			}
		}
		dnsserver2 := a.servers[1].Text
		dnsserver2 = strings.Trim(dnsserver2, " ")
		if len(dnsserver2) > 0 {
			if nh_util.NH_is_proper_ip(dnsserver2) {
				if len(dnsservers) == 0 {
					dnsservers = dnsserver2
				} else {
					dnsservers = dnsservers + "," + dnsserver2
				}
			} else {
				a.m.ShowAlert("Invalid DNS Server address " + dnsserver2)
				return
			}
		}
	}
	config.Vpnmode = a.curvpnmode
	config.Rentries = routes
	config.Servers = []DNSEntry{}
	config.DnsServers = dnsservers

	jc := m{
		"configsave": config,
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		a.m.ShowAlert("Nearhop Admin, Error while saving. Invalid Data")
		return
	}
	_, err = CommandCallback(SaveConfig, jsonData, len(jsonData))
	if err != nil {
		a.m.ShowAlert(err.Error())
	} else {
		a.m.ShowAlert("Config Saved. Close this Window and start again")
	}
}

func (a *AdminScreen) createObjects(objects []fyne.CanvasObject, height_offset float32, rname string, cidr string, via string, mtu int, i int) float32 {
	a.subnets[i].rname = NewNHEntryWithPlaceHolder("Enter Name", func() {
		a.saveConfig()
	})
	objects[i*number_of_items_in_subnet] = a.subnets[i].rname

	a.subnets[i].subnet = NewNHEntryWithPlaceHolder("Enter Subnet", func() {
		a.saveConfig()
	})
	objects[i*number_of_items_in_subnet+1] = a.subnets[i].subnet

	a.subnets[i].via = NewNHEntryWithPlaceHolder("Enter VIA IP Address", func() {
		a.saveConfig()
	})
	objects[i*number_of_items_in_subnet+2] = a.subnets[i].via

	a.subnets[i].mtu = NewNHEntryWithPlaceHolder("Enter MTU", func() {
		a.saveConfig()
	})
	objects[i*number_of_items_in_subnet+3] = a.subnets[i].mtu

	a.subnets[i].deleteCheck = NewNHLabelButton("  ", func() {
		dialog.ShowConfirm("Confirm", "Delete route "+a.subnets[i].rname.Text+"?", func(confirm bool) {
			if confirm {
				a.subnets[i].markedForDeletion = true
				a.saveConfig()
			}
		}, a.m.w)
	}, nil)
	objects[i*number_of_items_in_subnet+5] = a.subnets[i].deleteCheck

	var local_height_offset float32
	local_height_offset = 2 * theme.Padding()

	// Place Name
	a.subnets[i].rname.Move(fyne.Position{route_name_width_offset, height_offset + local_height_offset})
	a.subnets[i].rname.Resize(fyne.NewSize(route_name_width, route_name_height))

	// Place subnet
	a.subnets[i].subnet.Move(fyne.Position{route_subnet_width_offset, height_offset + local_height_offset})
	a.subnets[i].subnet.Resize(fyne.NewSize(route_subnet_width, route_subnet_height))

	local_height_offset += route_name_height + theme.Padding()

	// Via
	a.subnets[i].via.Move(fyne.Position{route_via_width_offset, height_offset + local_height_offset})
	a.subnets[i].via.Resize(fyne.NewSize(route_via_width, route_via_height))

	// MTU
	a.subnets[i].mtu.Move(fyne.Position{route_mtu_width_offset, height_offset + local_height_offset})
	a.subnets[i].mtu.Resize(fyne.NewSize(route_mtu_width, route_subnet_height))

	// Delete Check
	//delete_button_bg := canvas.NewImageFromResource(resourceDelete)
	delete_button_bg := canvas.NewImageFromResource(theme.DeleteIcon())
	delete_button_bg.Resize(fyne.NewSize(route_delete_bg_width, route_delete_bg_height))
	delete_button_bg.Move(fyne.Position{route_delete_bg_width_offset, height_offset + local_height_offset + 2*theme.Padding()})
	a.subnets[i].deleteCheck.Move(fyne.Position{route_deletecheck_width_offset, height_offset + local_height_offset + 2*theme.Padding()})
	a.subnets[i].deleteCheck.Resize(fyne.NewSize(route_deletecheck_width, route_deletecheck_height))
	objects[i*number_of_items_in_subnet+4] = delete_button_bg
	local_height_offset += route_name_height + 2*theme.Padding()

	//outline := canvas.NewRectangle(color.NRGBA{R: 96, G: 161, B: 193, A: 255})
	outline := &canvas.Rectangle{}
	outline.StrokeWidth = 1
	outline.StrokeColor = color.NRGBA{R: 0xA6, G: 0xDF, B: 0xF0, A: 255}
	outline.Resize(fyne.NewSize(outline_width, local_height_offset-2*theme.Padding()))
	outline.Move(fyne.Position{outline_width_offset, height_offset + theme.Padding()})
	objects[i*number_of_items_in_subnet+6] = outline

	a.subnets[i].rname.SetText(rname)
	a.subnets[i].subnet.SetText(cidr)
	a.subnets[i].via.SetText(via)
	a.subnets[i].mtu.SetText(strconv.Itoa(mtu))
	return float32(local_height_offset)
}

func (a *AdminScreen) populateCustomRoutes(c *config.C, newentry bool, wificlientsroutes bool) (error, float32) {
	numof_wificlientsroutes := 0
	var iplist IPList
	if wificlientsroutes {
		if a.wifirouterip == "" {
			gw, _ := CommandCallback(GetRouterIP, nil, 0)
			a.wifirouterip = string(gw)
		}
		if a.wifirouterip == "" {
			a.m.ShowAlert("No WiFi Router found. Make sure your WiFi router is onboarded")
			return fmt.Errorf("No WiFi Router found. Make sure your WiFi router is onboarded"), 0
		}
		a.m.ShowAlert("Fetching the client list. This may take upto 30 seconds. Please wait.")
		ret, err := CommandCallback(GetWiFiClientIPList, nil, 0)
		if err != nil {
			a.m.ShowAlert("Error while fetching the client list")
		} else {
			err = json.Unmarshal(ret, &iplist)
			if err != nil {
				a.m.ShowAlert("Error while unmarshalling the client list " + string(ret))
			}
			numof_wificlientsroutes = len(iplist.IPList)
		}
	}
	routes, err := overlay.ParseUnsafeRoutes(c, nil)
	if err != nil {
		return fmt.Errorf("Error while parsing unsafe routes", err), 0
	}
	a.subnetform = &fyne.Container{Layout: NewNearhopLayout()}
	//a.subnetform = &fyne.Container{Layout: layout.NewGridLayoutWithColumns(2)}
	if a.subnetform == nil {
		return fmt.Errorf("Error while creating Grid Container"), 0
	}

	entries_len := len(routes)
	entries_len += numof_wificlientsroutes
	if newentry {
		entries_len++
	}
	objects := make([]fyne.CanvasObject, number_of_items_in_subnet*entries_len)
	if objects == nil {
		return fmt.Errorf("Error while allocating memory for unsafe routes in UI"), 0
	}
	a.num_of_routes = 0
	var subnet_height float32

	subnet_height = 0
	for i, r := range routes {
		subnet_height += a.createObjects(objects, a.subnet_height_offset+subnet_height, r.Name, r.Cidr.String(), r.Via.String(), r.MTU, i)
		a.num_of_routes++
	}
	if wificlientsroutes {
		for i := 0; i < numof_wificlientsroutes; i++ {
			subnet_height += a.createObjects(objects, a.subnet_height_offset+subnet_height, iplist.IPList[i].Name, iplist.IPList[i].IPAddress+"/32", a.wifirouterip, 1300, a.num_of_routes)
			a.num_of_routes++
		}
	}

	if (newentry || wificlientsroutes) && a.num_of_routes >= MAX_SUBNETS {
		a.m.ShowAlert("Max number of routes already reached. Can't add more " + strconv.Itoa(MAX_SUBNETS))
	} else if newentry {
		subnet_height += a.createObjects(objects, a.subnet_height_offset+subnet_height, "", "", "", 1300, a.num_of_routes)
	}
	a.subnetform.Objects = append(a.subnetform.Objects, objects...)

	return nil, subnet_height
}

func (a *AdminScreen) populateFullVPNRoute(c *config.C) error {
	routes, err := overlay.ParseUnsafeRoutes(c, nil)
	if err != nil {
		return fmt.Errorf("Error while parsing unsafe routes", err)
	}

	//a.fullvpnform = &fyne.Container{Layout: layout.NewGridLayoutWithColumns(2)}
	a.fullvpnform = &fyne.Container{Layout: layout.NewVBoxLayout()}
	if a.fullvpnform == nil {
		return fmt.Errorf("Error while creating Grid Container")
	}

	objects := make([]fyne.CanvasObject, 2)
	if objects == nil {
		return fmt.Errorf("Error while allocating memory for Gateway IP in UI")
	}

	objects[0] = newLabel("Gateway VPN IP Address ", a.formLabelColor, theme.TextSize(), fyne.TextStyle{})
	a.subnets[0].via = NewNHEntry(func() {
		a.saveConfig()
	})
	objects[1] = a.subnets[0].via

	a.fullvpnform.Objects = append(a.fullvpnform.Objects, objects...)
	if routes != nil && len(routes) > 0 {
		a.subnets[0].via.SetText(routes[0].Via.String())
	} else {
		a.subnets[0].via.SetText(string(a.wifirouterip))
	}
	a.fullvpnform.Refresh()

	return nil
}

func (a *AdminScreen) populateDNSServers(c *config.C) error {
	dnsservers, err := overlay.GetDNSServers(c)
	if err != nil {
		return fmt.Errorf("Error while parsing unsafe routes", err)
	}

	a.dnsserverform = &fyne.Container{Layout: layout.NewVBoxLayout()}
	if a.dnsserverform == nil {
		return fmt.Errorf("Error while creating Grid Container")
	}

	objects := make([]fyne.CanvasObject, 4)
	if objects == nil {
		return fmt.Errorf("Error while allocating memory for Gateway IP in UI")
	}

	objects[0] = newLabel("DNS Server 1", a.formLabelColor, theme.TextSize(), fyne.TextStyle{})
	a.servers[0] = NewNHEntry(func() {
		a.saveConfig()
	})
	objects[1] = a.servers[0]
	if len(dnsservers) > 0 {
		a.servers[0].SetText(dnsservers[0].String())
	} else {
		a.servers[0].SetText(string(a.wifirouterip))
	}

	objects[2] = newLabel("DNS Server 2", a.formLabelColor, theme.TextSize(), fyne.TextStyle{})
	a.servers[1] = NewNHEntry(func() {
		a.saveConfig()
	})
	objects[3] = a.servers[1]
	if len(dnsservers) > 1 {
		a.servers[1].SetText(dnsservers[1].String())
	}
	a.dnsserverform.Objects = append(a.dnsserverform.Objects, objects...)

	return nil
}

func (a *AdminScreen) showDNS(show bool) {
	if show {
		if a.curvpnmode == CustomVPN || a.curvpnmode == FullVPN {
			a.dnsserverform.Show()
		} else {
			a.dnsserverform.Hide()
		}
	} else {
		a.dnsserverform.Hide()
	}
}

func (a *AdminScreen) resetClicked() {
	a.resetSubmit(a.m.w)
}

func (a *AdminScreen) newRouteClicked() {
	a.newentry = true
	a.m.ReloadAdmin()
}

func (a *AdminScreen) homeRoutesClicked() {
	a.wificlientsroutes = true
	a.m.ReloadAdmin()
}

func NewAdminScreen(m *MainWindow, c *config.C) *AdminScreen {
	adminScreen := &AdminScreen{}
	adminScreen.formLabelColor = color.NRGBA{R: 01, G: 07, B: 31, A: 255}
	adminScreen.config = c
	adminScreen.m = m
	reset := NewNHLabelButton("RESET", adminScreen.resetClicked, adminScreen.resetClicked)

	save := NewNHLabelButton("SAVE", adminScreen.saveConfig, nil)
	add := NewNHLabelButton("                ", adminScreen.newRouteClicked, nil)
	adminScreen.add = add

	adminScreen.clientsRoutes = NewNHLabelButton("ADD ROUTES TO MY HOME NETWORK",
		adminScreen.homeRoutesClicked, adminScreen.homeRoutesClicked)

	adminScreen.subnetform = nil
	adminScreen.fullvpnform = nil
	adminScreen.subnet_height_offset = 263.68

	form := &fyne.Container{Layout: layout.NewGridLayoutWithColumns(1)}
	objects := make([]fyne.CanvasObject, 1)
	adminScreen.radioGroup = widget.NewRadioGroup([]string{SemiVPN, FullVPN, CustomVPN}, func(clicked string) {
		adminScreen.curvpnmode = clicked
		if clicked == CustomVPN {
			adminScreen.showDNS(true)
		} else if clicked == FullVPN {
			adminScreen.showDNS(true)
		} else if clicked == SemiVPN {
			adminScreen.showDNS(false)
		}
		adminScreen.m.ReloadAdmin()
	})
	adminScreen.radioGroup.Horizontal = true

	gw, _ := CommandCallback(GetRouterIP, nil, 0)
	adminScreen.wifirouterip = string(gw)

	err := adminScreen.populateFullVPNRoute(c)
	if err != nil {
		adminScreen.m.ShowAlert(err.Error())
		return nil
	}
	err, _ = adminScreen.populateCustomRoutes(c, false, false)
	if err != nil {
		adminScreen.m.ShowAlert(err.Error())
		return nil
	}

	err = adminScreen.populateDNSServers(c)
	if err != nil {
		adminScreen.m.ShowAlert(err.Error())
		return nil
	}
	// Check the vpn mode and populate things accordingly
	full_vpn := c.GetBool("tun.fullvpn", false)
	if full_vpn {
		adminScreen.radioGroup.SetSelected(FullVPN)
		adminScreen.curvpnmode = FullVPN
		adminScreen.showDNS(true)
	} else {
		routes, err := overlay.ParseUnsafeRoutes(c, nil)
		if err != nil {
			adminScreen.m.ShowAlert("Error while parsing unsafe routes " + err.Error())
			return nil
		}
		if routes == nil || len(routes) == 0 {
			adminScreen.radioGroup.SetSelected(SemiVPN)
			adminScreen.curvpnmode = SemiVPN
			adminScreen.showDNS(false)
		} else {
			adminScreen.radioGroup.SetSelected(CustomVPN)
			adminScreen.curvpnmode = CustomVPN
			adminScreen.showDNS(true)
		}
	}
	objects[0] = adminScreen.radioGroup
	form.Objects = append(form.Objects, objects...)

	adminScreen.form = form
	adminScreen.reset = reset
	adminScreen.save = save
	adminScreen.add = add

	return adminScreen
}

func (a *AdminScreen) getFullVPNFormHeight() float32 {
	return a.fullvpnform.MinSize().Height
}

func (a *AdminScreen) getDNSServerFormHeight() float32 {
	return a.dnsserverform.MinSize().Height
}

func (a *AdminScreen) getSubnetFormHeight() float32 {
	return a.subnetform.MinSize().Height
}

func (a *AdminScreen) getNewRouteHeight() float32 {
	return a.add.MinSize().Height
}

func (a *AdminScreen) LayoutCustomRoutes(height_offset float32) float32 {
	err, offset := a.populateCustomRoutes(a.config, a.newentry, a.wificlientsroutes)
	if err != nil {
		a.m.ShowAlert(err.Error())
		return 0
	}
	return offset
}

func (a *AdminScreen) Show() fyne.CanvasObject {
	var height_offset float32
	var add_button_icon *canvas.Image

	describe := newLabel("Change VPN Mode", color.NRGBA{R: 00, G: 00, B: 00, A: 255}, 16, fyne.TextStyle{})
	describe.Move(fyne.Position{describe_width_offset, describe_height_offset})
	describe.Resize(fyne.NewSize(describe_width, describe_height))

	mode_selector := a.radioGroup
	mode_selector.Move(fyne.Position{mode_selector_width_offset, mode_selector_height_offset})
	mode_selector.Resize(fyne.NewSize(mode_selector_width, mode_selector_height))

	add_button_icon = nil
	height_offset = form_height_offset
	if a.curvpnmode == FullVPN {
		a.fullvpnform.Resize(fyne.NewSize(form_width, float32(a.getFullVPNFormHeight())))
		a.fullvpnform.Move(fyne.Position{form_width_offset, float32(height_offset)})
		height_offset += theme.Padding() + a.getFullVPNFormHeight()
		a.dnsserverform.Resize(fyne.NewSize(form_width, float32(a.getDNSServerFormHeight())))
		a.dnsserverform.Move(fyne.Position{form_width_offset, float32(height_offset)})
		height_offset += theme.Padding() + a.getDNSServerFormHeight()
	} else if a.curvpnmode == CustomVPN {
		a.dnsserverform.Resize(fyne.NewSize(form_width, float32(a.getDNSServerFormHeight())))
		a.dnsserverform.Move(fyne.Position{form_width_offset, float32(height_offset)})
		height_offset += theme.Padding() + a.getDNSServerFormHeight()

		height_offset += theme.Padding() * 2

		a.subnetform.Move(fyne.Position{form_width_offset, float32(a.subnet_height_offset)})
		local_offset := a.LayoutCustomRoutes(height_offset)
		a.subnetform.Resize(fyne.NewSize(form_width, height_offset+local_offset))

		height_offset += theme.Padding() + a.getSubnetFormHeight()

		add_button_icon = canvas.NewImageFromResource(resourceNewroute)
		add_button_icon.Resize(fyne.NewSize(newroute_icon_width, newroute_icon_height))
		add_button_icon.Move(fyne.Position{newroute_icon_width_offset, float32(height_offset + 10)})
		a.add.Resize(fyne.NewSize(newroute_text_width, newroute_text_width))
		a.add.Move(fyne.Position{newroute_text_width_offset, float32(height_offset)})
		height_offset += theme.Padding() + a.getNewRouteHeight()
	}
	height_offset += theme.Padding() * 2
	reset_button_bg := canvas.NewImageFromResource(resourceButtonLight)
	reset_button_bg.Resize(fyne.NewSize(reset_bg_width, reset_bg_height))
	reset_button_bg.Move(fyne.Position{reset_bg_width_offset, float32(height_offset)})
	a.reset.Resize(fyne.NewSize(reset_button_width, reset_button_height))
	a.reset.Move(fyne.Position{reset_button_width_offset, float32(height_offset)})

	save_button_bg := canvas.NewImageFromResource(resourceButtonDark)
	save_button_bg.Resize(fyne.NewSize(save_bg_width, save_bg_height))
	save_button_bg.Move(fyne.Position{save_bg_width_offset, float32(height_offset)})
	a.save.Resize(fyne.NewSize(save_button_width, save_button_height))
	a.save.Move(fyne.Position{save_button_width_offset, float32(height_offset)})

	clientsRoutes_button_bg := canvas.NewImageFromResource(resourceButtonLight)
	clientsRoutes_button_bg.Resize(fyne.NewSize(clientsroutes_bg_width, clientsroutes_bg_height))
	clientsRoutes_button_bg.Move(fyne.Position{clientsroutes_bg_width_offset, float32(height_offset)})
	a.clientsRoutes.Resize(fyne.NewSize(clientsroutes_button_width, clientsroutes_button_height))
	a.clientsRoutes.Move(fyne.Position{clientsroutes_button_width_offset, float32(height_offset)})

	right_panel_bg := canvas.NewRectangle(color.NRGBA{R: 96, G: 161, B: 193, A: 255})
	right_panel_bg.Resize(fyne.NewSize(window_width-left_panel_width, height_offset+reset_bg_height+2*theme.Padding()))
	right_panel_bg.Move(fyne.Position{float32(left_panel_width), 0})
	right_panel_bg.SetMinSize(fyne.NewSize(window_width-left_panel_width, height_offset+reset_bg_height+2*theme.Padding()))
	if a.curvpnmode == CustomVPN {
		return container.New(NewNearhopLayout(), right_panel_bg, describe, mode_selector, a.dnsserverform, a.subnetform, add_button_icon, a.add, reset_button_bg, a.reset, save_button_bg, a.save, clientsRoutes_button_bg, a.clientsRoutes)
	} else if a.curvpnmode == FullVPN {
		return container.New(NewNearhopLayout(), right_panel_bg, describe, mode_selector, a.fullvpnform, a.dnsserverform, reset_button_bg, a.reset, save_button_bg, a.save)
	} else {
		return container.New(NewNearhopLayout(), right_panel_bg, describe, mode_selector, reset_button_bg, a.reset, save_button_bg, a.save)
	}
}
