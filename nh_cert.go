package nebula

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/slackhq/nebula/cert"
	"golang.org/x/crypto/curve25519"
)

type ServerEntry struct {
	ServerPvtIP string `json:serverPvtIP`
	ServerIP    string `json:serverIP`
	Port        string `json:port`
	Id          string `json:_id`
}

type Certs struct {
	ErrorMessage string        `json:errorMessage`
	Ca           string        `json:ca`
	Cert         string        `json:cert`
	DeviceIp     string        `json:deviceIp`
	Servers      []ServerEntry `jsong:servers`
	Token        string        `json:token`
	DeviceId     string        `json:deviceId`
}

type SignMessage struct {
	Status  int   `json:status`
	Message Certs `json:message`
}

type SignErrorMessage struct {
	Status  int    `json:status`
	Message string `json:message`
}

func parseSignResponse(data []byte) (*SignMessage, error) {
	var signMessage SignMessage

	err := json.Unmarshal(data, &signMessage)
	if err != nil {
		var signErrorMessage SignErrorMessage
		err := json.Unmarshal(data, &signErrorMessage)

		if err != nil {
			return nil, err
		}
		signMessage.Status = signErrorMessage.Status
		signMessage.Message.ErrorMessage = signErrorMessage.Message
	}

	return &signMessage, nil
}

func open_mysql(sqlsecret string) (*sql.DB, error) {
	str := "root:" + sqlsecret + "@tcp(127.0.0.1:3306)/nearhop"
	db, err := sql.Open("mysql", str)
	if err != nil {
		return nil, err
	}
	return db, err
}

func shallSendSignRequest(f *Interface, networkID uint64) bool {
	lastSignRequest := f.signRequest[networkID]
	curTime := time.Now().Unix()
	sendSignRequest := false
	if lastSignRequest == 0 {
		sendSignRequest = true
	} else if curTime-lastSignRequest > 5 {
		sendSignRequest = true
	}
	return sendSignRequest
}

func X25519Keypair() ([]byte, []byte) {
	privkey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, privkey); err != nil {
		panic(err)
	}

	pubkey, err := curve25519.X25519(privkey, curve25519.Basepoint)
	if err != nil {
		panic(err)
	}

	return pubkey, privkey
}

func generate_and_sign_nh_certs(nwid uint64, relayIndex byte, sqlsecret string, keysecret string) (string, string, string, error) {
	pub, priv := X25519Keypair()

	publicKey := string(cert.MarshalX25519PublicKey(pub))
	privateKey := string(cert.MarshalX25519PrivateKey(priv))

	data, err := nh_http_sign_server_certs(nwid, publicKey, relayIndex, keysecret)
	if err != nil {
		return "", "", "", err
	}
	signMessage, err := parseSignResponse(data)
	if err != nil {
		return "", "", "", err
	}
	certs := &signMessage.Message
	if signMessage.Status == 0 {
		return "", "", "", fmt.Errorf(string(signMessage.Message.ErrorMessage))
	}

	nwidstr := strconv.FormatUint(nwid, 10)
	db, err := open_mysql(sqlsecret)
	if err != nil {
		return "", "", "", err
	}
	defer db.Close()

	sql := "INSERT INTO certs(networkid, cacrt, certkey, certcrt)  VALUES (\"" + nwidstr + "\",\"" + certs.Ca + "\",\"" + privateKey + "\",\"" + certs.Cert + "\")"
	_, err = db.Exec(sql)
	if err != nil {
		return "", "", "", err
	}

	return certs.Ca, privateKey, certs.Cert, err
}

func Sign_nh_client_certs(email string, key string, name string, publicKey string) (*Certs, error) {
	data, err := nh_http_sign_client_certs(email, key, name, publicKey)
	if err != nil {
		return nil, err
	}
	signMessage, err := parseSignResponse(data)
	if err != nil {
		return nil, err
	}
	certs := &signMessage.Message
	return certs, nil
}

func get_network_certs(nwid uint64, sqlsecret string) (string, string, string, error) {
	nwidstr := strconv.FormatUint(nwid, 10)
	db, err := open_mysql(sqlsecret)
	if err != nil {
		return "", "", "", err
	}
	defer db.Close()

	query := "select cacrt, certkey, certcrt from certs where networkid=" + nwidstr
	res, err := db.Query(query)
	if err != nil {
		return "", "", "", err
	}
	defer res.Close()

	for res.Next() {
		var cacrt string
		var certkey string
		var certcrt string

		err := res.Scan(&cacrt, &certkey, &certcrt)
		if err != nil {
			return "", "", "", err
		}
		return cacrt, certkey, certcrt, nil
	}

	return "", "", "", fmt.Errorf("Unknown error while fetching certkey, certcrt from database")
}
