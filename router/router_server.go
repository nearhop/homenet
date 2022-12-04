//go:build router
// +build router

package router

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	messages "messages"
	nh_util "nh_util"

	"github.com/sirupsen/logrus"
)

type mp map[string]interface{}

type RouterServer struct {
	tel        *Telemetry
	l          *logrus.Logger
	ctx        context.Context
	uploadlogs bool
}

func NewRouterServer(l1 *logrus.Logger) (*RouterServer, error) {
	rs := &RouterServer{l: l1}
	rs.ctx, _ = context.WithCancel(context.Background())
	rs.tel = NewTelemetry(l1)
	go rs.tel.Run(rs.ctx)
	return rs, nil
}

func (rs *RouterServer) telemetry(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var telemetryData TelemetryData
		body, _ := ioutil.ReadAll(r.Body) // check for errors
		err := json.Unmarshal(body, &telemetryData)
		if err != nil {
			rs.l.Error("Error while unmarshalling telemetry", err)
			return
		}
		rs.tel.processTelemetry(&telemetryData)
	}
	status := "{\"status\": \"success\"}"
	fmt.Fprintf(w, status)
}

func (rs *RouterServer) wireless(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var wirelessTelemetryData WirelessTelemetryData
		body, _ := ioutil.ReadAll(r.Body) // check for errors
		err := json.Unmarshal(body, &wirelessTelemetryData)
		if err != nil {
			rs.l.Error("Error while unmarshalling wireless telemetry", err)
			return
		}
		rs.tel.processWirelessTelemetry(&wirelessTelemetryData)
	}
	status := "{\"status\": \"success\"}"
	fmt.Fprintf(w, status)
}

func (rs *RouterServer) processCommand(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		inmsg, _ := ioutil.ReadAll(r.Body) // check for errors
		ret := messages.ProcessMessage(inmsg, rs.l, nil)
		fmt.Fprintf(w, string(ret))
	}
}

func (rs *RouterServer) dumpClients(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		clients, err := rs.tel.dumpClientsJson()
		if err == nil {
			fmt.Fprintf(w, string(clients))
		} else {
			rs.l.Error("Error while dumping clients", err)
			fmt.Fprintf(w, nh_util.NH_getErrorStatusString(err.Error()))
		}
	}
}

func (rs *RouterServer) pauseClient(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var pauseMessage messages.PauseMessage
		body, _ := ioutil.ReadAll(r.Body) // check for errors
		err := json.Unmarshal(body, &pauseMessage)
		if err != nil {
			rs.l.Error("Error while unmarshalling Pause Message", err)
			fmt.Fprintf(w, nh_util.NH_getErrorStatusString(err.Error()))
			return
		}
		var output string

		status := "success"
		if pauseMessage.Mbody.MACAddress == "all" {
			output = rs.tel.pauseAll(pauseMessage.Mbody.Pause)
		} else {
			output = rs.tel.pauseClient(pauseMessage.Mbody.MACAddress, pauseMessage.Mbody.Pause)
		}
		if output == "na" {
			rs.l.Error("Error while pausing client(s)", pauseMessage.Mbody.MACAddress)
			status = "fail"
		}
		jc := mp{
			"status":     status,
			"MACAddress": pauseMessage.Mbody.MACAddress,
			"Pause":      pauseMessage.Mbody.Pause,
		}

		jsonData, err := json.Marshal(jc)
		status = string(jsonData)
		fmt.Fprintf(w, status)
	}
}

func (rs *RouterServer) setClientDetails(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		var clientMessage messages.ClientMessage
		body, _ := ioutil.ReadAll(r.Body) // check for errors
		err := json.Unmarshal(body, &clientMessage)
		if err != nil {
			rs.l.Error("Error while unmarshalling client Message", err)
			fmt.Fprintf(w, nh_util.NH_getErrorStatusString(err.Error()))
			return
		}
		output := rs.tel.setClientDetails(clientMessage.Mbody.MACAddress, clientMessage.Mbody.Name, clientMessage.Mbody.Type)
		status := "success"
		if output != "" {
			status = output
		}
		jc := mp{
			"status":     status,
			"MACAddress": clientMessage.Mbody.MACAddress,
			"Name":       clientMessage.Mbody.Name,
		}

		jsonData, err := json.Marshal(jc)
		status = string(jsonData)
		fmt.Fprintf(w, status)
	}
}

func (rs *RouterServer) registerRepeater(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		body, _ := ioutil.ReadAll(r.Body) // check for errors

		err := rs.tel.registerRepeater(body)
		if err != nil {
			fmt.Fprintf(w, nh_util.NH_getErrorStatusString(err.Error()))
			return
		}
		fmt.Fprintf(w, "{\"status\":\"success\"}")
	}
}

func (rs *RouterServer) getRepeaters(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		clients, err := rs.tel.RepeatersJson()
		if err == nil {
			fmt.Fprintf(w, string(clients))
		} else {
			rs.l.Error("Error while dumping clients", err)
			fmt.Fprintf(w, nh_util.NH_getErrorStatusString(err.Error()))
		}
	}
}

func (rs *RouterServer) isOnboardingOpen(w http.ResponseWriter, r *http.Request) {
	cmd := "/sbin/uci"
	args := []string{"get", "wireless.onboard.device"}
	out := nh_util.NH_read_cmd_output(cmd, args)
	if out == "" || out == "na" {
		out = "NO"
	} else {
		out = "YES"
	}
	fmt.Fprintf(w, out)
}

func (rs *RouterServer) uploadLogs(w http.ResponseWriter, r *http.Request) {
	rs.uploadlogs = true
}

func (rs *RouterServer) ShallUploadLogs() bool {
	if rs.uploadlogs {
		rs.uploadlogs = false
		return true
	} else {
		return false
	}
}

func (rs *RouterServer) router_onboard_status(w http.ResponseWriter, r *http.Request) {
	status := "{\"onboarded\": \"true\"}"
	fmt.Fprintf(w, status)
}

func (rs *RouterServer) GetNextEvent() *RouterEvent {
	return rs.tel.getNextEvent()
}

func (rs *RouterServer) MarkRouterEvent(event *RouterEvent, value bool) {
	event.Active = value
}

func (rs *RouterServer) GetRouterEventMessage(event *RouterEvent) ([]byte, error) {
	jc := mp{
		"etype":      event.Etype,
		"ipaddress":  event.Client.IPAddress,
		"macAddress": event.Client.MACAddress,
		"name":       event.Client.Name,
		"extra":      event.Extra,
		"tstamp":     event.Tstamp,
	}
	jc1 := mp{
		"type":  "router_event",
		"Mbody": jc,
	}

	jsonData, err := json.Marshal(jc1)
	if err != nil {
		rs.l.Error("handleEvent: Error while marshalling data " + err.Error())
		return nil, err
	}
	return jsonData, nil
}

func (rs *RouterServer) StartRouterServer() error {
	http.HandleFunc("/telemetry", rs.telemetry)
	http.HandleFunc("/radios", rs.wireless)
	http.HandleFunc("/command", rs.processCommand)
	http.HandleFunc("/clients", rs.dumpClients)
	http.HandleFunc("/setclientdetails", rs.setClientDetails)
	http.HandleFunc("/pause", rs.pauseClient)
	http.HandleFunc("/unpause", rs.pauseClient)
	http.HandleFunc("/pauseall", rs.pauseClient)
	http.HandleFunc("/unpauseall", rs.pauseClient)
	http.HandleFunc("/register_repeater", rs.registerRepeater)
	http.HandleFunc("/repeaters", rs.getRepeaters)
	http.HandleFunc("/isonboardingopen", rs.isOnboardingOpen)
	http.HandleFunc("/router_onboard_status", rs.router_onboard_status)
	http.HandleFunc("/uploadlogs", rs.uploadLogs)

	rs.l.Fatal(http.ListenAndServe("0.0.0.0:11000", nil))
	rs.l.Info("SServer started at port 11000\n")

	clockSource := time.NewTicker(5 * time.Second)
	defer clockSource.Stop()
	for {
		select {
		case <-rs.ctx.Done():
			return nil
		case _ = <-clockSource.C:
		}
	}
	return nil
}
