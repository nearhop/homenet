package nebula

import (
	"encoding/json"
	"strconv"

	nh_util "nh_util"
	platform "platform"

	"github.com/denisbrodbeck/machineid"
)

func nh_http_sign_server_certs(nwid uint64, pubKey string, relayIndex byte, keysecret string) ([]byte, error) {
	var nwidStr = strconv.FormatUint(nwid, 10)
	jc := m{
		"nwid":       nwidStr,
		"pubKey":     pubKey,
		"relayIndex": relayIndex,
		"apiKey":     keysecret,
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		return nil, err
	}
	bytes, err, _ := nh_util.Nh_http_send_req(nh_util.Homeneturl+"hnoapi/signServerPubKey", jsonData)
	return bytes, err
}

func nh_http_sign_client_certs(email string, key string, name string, pubKey string) ([]byte, error) {
	id, err := machineid.ID()
	if err != nil {
		id = platform.Get_deviceid()
	}
	jc := m{
		"deviceId":   id,
		"deviceName": name,
		"email":      email,
		"stkey":      key,
		"pubKey":     pubKey,
		"deviceOS":   platform.Get_osname(),
		"devFunType": platform.Get_Device_Type(),
	}

	jsonData, err := json.Marshal(jc)
	if err != nil {
		return nil, err
	}
	bytes, err, _ := nh_util.Nh_http_send_req(nh_util.Homeneturl+"hnoapi/signClientPubKey", jsonData)
	return bytes, err
}
