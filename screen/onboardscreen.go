//go:build screen && !server
// +build screen,!server

package screen

import (
	"encoding/json"
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
)

const onboard_window_width = 878
const onboard_window_height = 488
const onboard_outline_width = 340
const onboard_outline_height = 380
const onboard_outline_width_offset = 52
const onboard_outline_height_offset = 52
const onboard_logo_width = 149
const onboard_logo_height = 25.26
const onboard_logo_width_offset = 134.68
const onboard_logo_height_offset = 110.4
const please_width = 188
const please_height = 24
const please_width_offset = 129
const please_height_offset = 162
const help_width = 167
const help_height = 14
const help_width_offset = 139
const help_height_offset = 186
const email_width = 290
const email_height = 40
const email_width_offset = 77
const email_height_offset = 217
const key_width = 290
const key_height = 40
const key_width_offset = 77
const key_height_offset = 272
const onboard_name_width = 290
const onboard_name_height = 40
const onboard_name_width_offset = 77
const onboard_name_height_offset = 327
const onboard_submit_width = 290
const onboard_submit_height = 40
const onboard_submit_width_offset = 70
const onboard_submit_height_offset = 382
const onboard_submit_bg_width = 290
const onboard_submit_bg_height = 40
const onboard_submit_bg_width_offset = 77
const onboard_submit_bg_height_offset = 382
const onboard_submit_text_width = 100
const onboard_submit_text_height = 40
const onboard_submit_text_width_offset = 195
const onboard_submit_text_height_offset = 382

type m map[string]interface{}

type OnboardScreen struct {
	email          *NHEntry
	key            *NHEntry
	name           *NHEntry
	form           *fyne.Container
	submit         *NHLabelButton
	mw             *MainWindow
	formLabelColor color.NRGBA
}

func (o *OnboardScreen) disableForm() {
	o.email.Disable()
	o.name.Disable()
	o.key.Disable()
}

func (o *OnboardScreen) enableForm() {
	o.email.Enable()
	o.name.Enable()
	o.key.Enable()
}

func (o *OnboardScreen) OnboardSubmit() {
	email := o.email.Text
	if len(email) == 0 {
		o.mw.ShowAlert("Email ID should n't be blank")
		return
	}
	name := o.name.Text
	if len(name) == 0 {
		o.mw.ShowAlert("Name should n't be blank")
		return
	}
	key := o.key.Text
	if len(key) == 0 {
		o.mw.ShowAlert("Key should n't be blank")
		return
	}
	jc := m{
		"email": email,
		"key":   key,
		"name":  name,
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		o.mw.ShowAlert("Cannot pass onboarding information to main")
		return
	}
	o.mw.ShowAlert("Please wait. Your device is being onboarded")
	o.disableForm()
	_, err = CommandCallback(OnboardClient, jsonData, len(jsonData))
	if err != nil {
		o.mw.ShowAlert("Error while onboarding device. Check your email/key and generate afresh from mobile app" + err.Error())
	} else {
		o.mw.ShowAlert("Congrats. Your device is onboarded")
	}
	o.enableForm()
}

func NewOnboardForm(onboardScreen *OnboardScreen) *fyne.Container {
	form := &fyne.Container{Layout: NewNearhopLayout()}

	objects := make([]fyne.CanvasObject, 3)
	onboardScreen.formLabelColor = color.NRGBA{R: 01, G: 07, B: 31, A: 255}

	onboardScreen.email = NewNHEntryWithPlaceHolder("Enter email ID", func() {
		onboardScreen.OnboardSubmit()
	})
	objects[0] = onboardScreen.email

	onboardScreen.key = NewNHEntryWithPlaceHolder("Enter Key(Get from mobile app)", func() {
		onboardScreen.OnboardSubmit()
	})
	objects[1] = onboardScreen.key

	onboardScreen.name = NewNHEntryWithPlaceHolder("Enter Name", func() {
		onboardScreen.OnboardSubmit()
	})
	objects[2] = onboardScreen.name

	onboardScreen.email.Resize(fyne.NewSize(email_width, email_height))
	onboardScreen.email.Move(fyne.Position{email_width_offset, email_height_offset})

	onboardScreen.key.Resize(fyne.NewSize(key_width, key_height))
	onboardScreen.key.Move(fyne.Position{key_width_offset, key_height_offset})

	onboardScreen.name.Resize(fyne.NewSize(onboard_name_width, onboard_name_height))
	onboardScreen.name.Move(fyne.Position{onboard_name_width_offset, onboard_name_height_offset})

	form.Objects = append(form.Objects, objects...)

	return form
}

func NewOnboardScreen(m *MainWindow, cert string) *OnboardScreen {
	onboardScreen := &OnboardScreen{}
	f := NewOnboardForm(onboardScreen)
	s := NewNHLabelButton("     ", onboardScreen.OnboardSubmit, onboardScreen.OnboardSubmit)

	onboardScreen.form = f
	onboardScreen.submit = s
	onboardScreen.mw = m

	return onboardScreen
}

func (o *OnboardScreen) Show() fyne.CanvasObject {
	onboard_window_bg := canvas.NewImageFromResource(resourceOnboardbackground)
	onboard_window_bg.Resize(fyne.NewSize(onboard_window_width, onboard_window_height))
	onboard_window_bg.Move(fyne.Position{0, 0})
	onboard_window_bg.SetMinSize(fyne.NewSize(onboard_window_width, onboard_window_height))

	outline := canvas.NewRectangle(color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff})
	outline.Resize(fyne.NewSize(onboard_outline_width, onboard_outline_height))
	outline.Move(fyne.Position{onboard_outline_width_offset, onboard_outline_height_offset})

	logo := canvas.NewImageFromResource(resourceLogobig)
	logo.Resize(fyne.NewSize(onboard_logo_width, onboard_logo_height))
	logo.Move(fyne.Position{onboard_logo_width_offset, onboard_logo_height_offset})

	please := newLabel("Please enter Email ID and Key", o.formLabelColor, 14, fyne.TextStyle{})
	please.Resize(fyne.NewSize(please_width, please_height))
	please.Move(fyne.Position{please_width_offset, please_height_offset})

	help := newLabel("(Get key from your mobile application)", o.formLabelColor, 10, fyne.TextStyle{})
	help.Resize(fyne.NewSize(help_width, help_height))
	help.Move(fyne.Position{help_width_offset, help_height_offset})

	submit_button_bg := canvas.NewImageFromResource(resourceButtonDark)
	submit_button_bg.Resize(fyne.NewSize(onboard_submit_bg_width, onboard_submit_bg_height))
	submittext := newLabel("VERIFY", color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff}, 14, fyne.TextStyle{Bold: true})
	submittext.Resize(fyne.NewSize(onboard_submit_text_width, onboard_submit_text_height))
	submittext.Move(fyne.Position{onboard_submit_text_width_offset, onboard_submit_text_height_offset})
	submit_button_bg.Move(fyne.Position{onboard_submit_bg_width_offset, onboard_submit_bg_height_offset})
	o.submit.Resize(fyne.NewSize(onboard_submit_width, onboard_submit_height))
	o.submit.Move(fyne.Position{onboard_submit_width_offset, onboard_submit_height_offset})

	return container.New(NewNearhopLayout(), onboard_window_bg, outline, logo, please, help, o.email, o.key, o.name, submit_button_bg, o.submit, submittext)
}
