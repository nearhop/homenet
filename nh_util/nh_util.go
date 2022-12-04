package util

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const Homeneturl = "https://homenet.nearhop.com/"

type mp map[string]interface{}

func Ip2int(ip []byte) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func NetIp2int(ip net.IP) uint32 {
	if len(ip) == 16 {
		return binary.BigEndian.Uint32(ip[12:16])
	}
	return binary.BigEndian.Uint32(ip)
}

func Int2ip(nn uint32) net.IP {
	ip := make(net.IP, net.IPv4len)
	binary.BigEndian.PutUint32(ip, nn)
	return ip
}

func NH_checksum(data []byte) uint32 {
	var csum uint32
	csum = 0
	// to handle odd lengths, we loop to length - 1, incrementing by 2, then
	// handle the last byte specifically by checking against the original
	// length.
	length := len(data) - 1
	for i := 0; i < length; i += 2 {
		// For our test packet, doing this manually is about 25% faster
		// (740 ns vs. 1000ns) than doing it by calling binary.BigEndian.Uint16.
		csum += uint32(data[i]) << 8
		csum += uint32(data[i+1])
	}
	if len(data)%2 == 1 {
		csum += uint32(data[length]) << 8
	}
	for csum > 0xffffffff {
		csum = (csum >> 32) + (csum & 0xffffffff)
	}
	return csum
}

func NH_read_cmd_output(cmd string, args []string) string {
	out, err := exec.Command(cmd, args...).Output()

	var sout string
	// not checking output. Just return the output na
	if out == nil || err != nil {
		sout = "na"
	} else {
		sout = string(out)
	}
	out1 := strings.TrimSuffix(string(sout), "\n")
	return out1
}

func NH_dump_to_file(filename string, bytes []byte, permissions fs.FileMode) error {
	return ioutil.WriteFile(filename, bytes, permissions)
}

func NH_create_dir(dirname string, permissions fs.FileMode) error {
	_, err := os.Stat(dirname)
	if err != nil {
		err = os.Mkdir(dirname, permissions)
		return err
	}
	return nil
}

func NH_read_file(filename string) ([]byte, error) {
	return ioutil.ReadFile(filename)
}

func NH_is_proper_subnet(subnet string) bool {
	_, _, err := net.ParseCIDR(subnet)
	if err == nil {
		return true
	} else {
		return false
	}
}

func NH_is_proper_ip(ipstr string) bool {
	ip := net.ParseIP(ipstr)
	if ip == nil {
		return false
	}
	return true
}

func NH_is_proper_Integer(svalue string, min int, max int) bool {
	value, err := strconv.Atoi(svalue)
	if err != nil {
		return false
	}
	if value < min || value > max {
		return false
	}
	return true
}

func NH_convert_into_xB(num uint64) string {
	units := ""
	var rem uint64

	rem = 0
	if num > (1024 * 1024 * 1024) {
		rem = num % (1024 * 1024 * 1024)
		num = num / (1024 * 1024 * 1024)
		units = "GB"
	} else if num > (1024 * 1024) {
		rem = num % (1024 * 1024)
		num = num / (1024 * 1024)
		units = "MB"
	} else if num > 1024 {
		rem = num % 1024
		num = num / 1024
		units = "KB"
	}
	remstr := strconv.FormatUint(rem, 10)
	numstr := strconv.FormatUint(num, 10)

	if len(remstr) > 1 {
		numstr = numstr + "." + remstr[0:2] + " " + units
	} else {
		numstr = numstr + "." + remstr + " " + units
	}
	return numstr
}

func NH_convert_into_xbps(num uint64, factor uint64) string {
	units := ""
	var rem uint64

	rem = 0
	num = num * 8      // Convert to bits
	num = num / factor // factor = period between two iterations
	if num > (1024 * 1024 * 1024) {
		rem = num % (1024 * 1024 * 1024)
		num = num / (1024 * 1024 * 1024)
		units = "Gbps"
	} else if num > (1024 * 1024) {
		rem = num % (1024 * 1024)
		num = num / (1024 * 1024)
		units = "Mbps"
	} else {
		rem = num % 1024
		num = num / 1024
		units = "Kbps"
	}
	remstr := strconv.FormatUint(rem, 10)
	numstr := strconv.FormatUint(num, 10)

	if len(remstr) > 1 {
		numstr = numstr + "." + remstr[0:2] + " " + units
	} else {
		numstr = numstr + "." + remstr + " " + units
	}
	return numstr
}

func NH_is_multicast_mac(mac string) (bool, error) {
	bs, err := hex.DecodeString(mac[0:2])
	if err != nil {
		return false, err
	}
	bytes := []byte(bs)
	if bytes[0]&0x01 == 0 {
		return false, nil
	} else {
		return true, nil
	}
}

func NH_http_get_req(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return body, nil
}

func Nh_http_send_req(url string, jsonData []byte) ([]byte, error, int) {
	request, error := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	request.Header.Set("Content-Type", "application/json; charset=UTF-8")

	client := &http.Client{}
	resp, error := client.Do(request)
	if error != nil {
		return nil, error, 0
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("http Error ", resp.StatusCode), resp.StatusCode
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	return bytes, err, resp.StatusCode
}

func NH_getErrorStatusString(errString string) string {
	jc := mp{
		"status": "fail",
		"error":  errString,
	}
	jsonData, err := json.Marshal(jc)
	if err == nil {
		return string(jsonData)
	} else {
		return "{\"status\": \"fail\"}"
	}
}

func NH_get_macaddress(iname string) (string, error) {
	ifas, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifa := range ifas {
		if iname == ifa.Name {
			return ifa.HardwareAddr.String(), nil
		}
	}
	return "", fmt.Errorf("Address Not found")
}

func NH_get_ipv4address(iname string) (string, error) {
	ifas, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, ifa := range ifas {
		if iname == ifa.Name {
			addrs, err := ifa.Addrs()
			if err != nil {
				return "", err
			}
			for _, addr := range addrs {
				ipv4Addr := addr.(*net.IPNet).IP.To4()
				if ipv4Addr == nil {
					continue
				}
				switch v := addr.(type) {
				case *net.IPNet:
					return v.IP.String(), nil
				case *net.IPAddr:
					return v.IP.String(), nil
				}
			}
		}
	}
	return "", fmt.Errorf("Address Not found")
}
