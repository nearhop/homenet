//go:build router
// +build router

package screen

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	router "router"

	"github.com/slackhq/nebula/config"
)

type mp map[string]interface{}

type MainWindow struct {
	onboarded bool
	routerip  string
}

var CommandCallback GUICommandCallback

var sign_status string
var observer *http.Server

func NewMainWindow() (*MainWindow, error) {
	m := &MainWindow{}
	m.onboarded = false
	return m, nil
}

func (m *MainWindow) signcert(w http.ResponseWriter, r *http.Request) {
	if m.onboarded {
		return
	}
	email := ""
	stkey := ""
	switch r.Method {
	case "POST":
		body, _ := ioutil.ReadAll(r.Body) // check for errors
		keyVal := make(map[string]string)
		json.Unmarshal(body, &keyVal) // check for errors
		email = keyVal["email"]
		stkey = keyVal["stkey"]
	}

	if email == "" || stkey == "" {
		status := "{\"status\": \"Missing email or key\"}"
		fmt.Fprintf(w, status)
		return
	}

	hostname, err := os.Hostname()
	if err != nil {
		hostname = "Router"
	}

	opmode := "gw"
	// Venkat: ToDo: Hanle appip
	jc := mp{
		"email": email,
		"key":   stkey,
		"name":  hostname,
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		status := "{\"status\": \"Cannot build sign request\"}"
		fmt.Fprintf(w, status)
		return
	}
	_, err = CommandCallback(OnboardClient, jsonData, len(jsonData))
	if err != nil {
		status := "{\"status\": \"Cannot sign certificates. Probably the email or key is wrong?\"}"
		fmt.Fprintf(w, status)
		return
	}
	jc = mp{
		"status":   "success",
		"routerip": m.routerip,
	}
	jsonData, err = json.Marshal(jc)
	if err != nil {
		status := "{\"status\": \"Onboarded. but error while sending the status to tyou\"}"
		fmt.Fprintf(w, status)
		return
	}
	go router.Router_onboarded(opmode)
	status := string(jsonData)
	fmt.Fprintf(w, status)
}

func (m *MainWindow) router_status(w http.ResponseWriter, r *http.Request) {
	var status string

	if m.onboarded {
		status = "{\"onboarded\": \"true\"}"
	} else {
		status = "{\"onboarded\": \"false\"}"
	}
	fmt.Fprintf(w, status)
}

func (m *MainWindow) StartMainWindow(onboarded bool, c *config.C, cert string, status_err string, callback GUICommandCallback) error {
	CommandCallback = callback
	m.onboarded = onboarded
	if !onboarded {
		sign_status = "Process not started"
		http.HandleFunc("/signcert", m.signcert)
		http.HandleFunc("/router_status", m.router_status)
		fmt.Printf("Server started at port 19001\n")
		http.ListenAndServe(":19001", nil)
	}

	ctx, _ := context.WithCancel(context.Background())
	clockSource := time.NewTicker(5 * time.Second)
	defer clockSource.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case _ = <-clockSource.C:
		}
	}
	return nil
}

func (m *MainWindow) SetHomeDetails(hd *HomeDetails) {
}

func (m *MainWindow) Onboarded(ip string) {
	m.onboarded = true
	m.routerip = ip
}
