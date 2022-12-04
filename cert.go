package nebula

import (
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/cert"
	"github.com/slackhq/nebula/config"
)

type CertState struct {
	certificate         *cert.NebulaCertificate
	rawCertificate      []byte
	rawCertificateNoKey []byte
	publicKey           []byte
	privateKey          []byte
}

func NewCertState(certificate *cert.NebulaCertificate, privateKey []byte) (*CertState, error) {
	// Marshal the certificate to ensure it is valid
	rawCertificate, err := certificate.Marshal()
	if err != nil {
		return nil, fmt.Errorf("invalid nebula certificate on interface: %s", err)
	}

	publicKey := certificate.Details.PublicKey
	cs := &CertState{
		rawCertificate: rawCertificate,
		certificate:    certificate, // PublicKey has been set to nil above
		privateKey:     privateKey,
		publicKey:      publicKey,
	}

	cs.certificate.Details.PublicKey = nil
	rawCertNoKey, err := cs.certificate.Marshal()
	if err != nil {
		return nil, fmt.Errorf("error marshalling certificate no key: %s", err)
	}
	cs.rawCertificateNoKey = rawCertNoKey
	// put public key back
	cs.certificate.Details.PublicKey = cs.publicKey
	return cs, nil
}

func NewCertStateFromFiles(privPathOrPEM string, pubPathOrPEM string) (*CertState, error) {
	var pemPrivateKey []byte
	var err error
	var rawCert []byte

	if strings.Contains(privPathOrPEM, "-----BEGIN") {
		pemPrivateKey = []byte(privPathOrPEM)
		privPathOrPEM = "<inline>"
	} else {
		pemPrivateKey, err = ioutil.ReadFile(privPathOrPEM)
		if err != nil {
			return nil, fmt.Errorf("unable to read pki.key file %s: %s", privPathOrPEM, err)
		}
	}

	if strings.Contains(pubPathOrPEM, "-----BEGIN") {
		rawCert = []byte(pubPathOrPEM)
		pubPathOrPEM = "<inline>"
	} else {
		rawCert, err = ioutil.ReadFile(pubPathOrPEM)
		if err != nil {
			return nil, fmt.Errorf("2. unable to read pki.cert file %s: %s", pubPathOrPEM, err)
		}
	}
	rawKey, _, err := cert.UnmarshalX25519PrivateKey(pemPrivateKey)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling pki.key %s: %s", privPathOrPEM, err)
	}

	nebulaCert, _, err := cert.UnmarshalNebulaCertificateFromPEM(rawCert)
	if err != nil {
		return nil, fmt.Errorf("error while unmarshaling pki.cert %s: %s", pubPathOrPEM, err)
	}

	/*
		if nebulaCert.Expired(time.Now()) {
			return nil, fmt.Errorf("nebula certificate for this host is expired")
		}
	*/

	if len(nebulaCert.Details.Ips) == 0 {
		return nil, fmt.Errorf("no IPs encoded in certificate")
	}

	if err = nebulaCert.VerifyPrivateKey(rawKey); err != nil {
		return nil, fmt.Errorf("private key is not a pair with public key in nebula cert")
	}

	return NewCertState(nebulaCert, rawKey)
}

func NewCertStateFromConfig(c *config.C) (*CertState, error) {
	privPathOrPEM := c.GetString("pki.key", "")
	if privPathOrPEM == "" {
		// Support backwards compat with the old x509
		//TODO: remove after this is rolled out everywhere - NB 2018/02/23
		privPathOrPEM = c.GetString("x509.key", "")
	}

	if privPathOrPEM == "" {
		return nil, errors.New("no pki.key path or PEM data provided")
	}

	pubPathOrPEM := c.GetString("pki.cert", "")
	if pubPathOrPEM == "" {
		// Support backwards compat with the old x509
		//TODO: remove after this is rolled out everywhere - NB 2018/02/23
		pubPathOrPEM = c.GetString("x509.cert", "")
	}

	if pubPathOrPEM == "" {
		return nil, errors.New("no pki.cert path or PEM data provided")
	}

	return NewCertStateFromFiles(privPathOrPEM, pubPathOrPEM)
}

func loadCAFromConfig(l *logrus.Logger, c *config.C) (*cert.NebulaCAPool, error) {
	var rawCA []byte
	var err error

	caPathOrPEM := c.GetString("pki.ca", "")
	if caPathOrPEM == "" {
		return nil, errors.New("no pki.ca path or PEM data provided")
	}

	if strings.Contains(caPathOrPEM, "-----BEGIN") {
		rawCA = []byte(caPathOrPEM)

	} else {
		rawCA, err = ioutil.ReadFile(caPathOrPEM)
		if err != nil {
			return nil, fmt.Errorf("2. unable to read pki.ca file %s: %s", caPathOrPEM, err)
		}
	}

	CAs, err := cert.NewCAPoolFromBytes(rawCA)
	if errors.Is(err, cert.ErrExpired) {
		var expired int
		for _, cert := range CAs.CAs {
			if cert.Expired(time.Now()) {
				expired++
				l.WithField("cert", cert).Warn("expired certificate present in CA pool")
			}
		}

		if expired >= len(CAs.CAs) {
			return nil, errors.New("no valid CA certificates present")
		}

	} else if err != nil {
		return nil, fmt.Errorf("error while adding CA certificate to CA trust store: %s", err)
	}

	for _, fp := range c.GetStringSlice("pki.blocklist", []string{}) {
		l.WithField("fingerprint", fp).Infof("Blocklisting cert")
		CAs.BlocklistFingerprint(fp)
	}

	// Support deprecated config for at least one minor release to allow for migrations
	//TODO: remove in 2022 or later
	for _, fp := range c.GetStringSlice("pki.blacklist", []string{}) {
		l.WithField("fingerprint", fp).Infof("Blocklisting cert")
		l.Warn("pki.blacklist is deprecated and will not be supported in a future release. Please migrate your config to use pki.blocklist")
		CAs.BlocklistFingerprint(fp)
	}

	return CAs, nil
}

func getCAFileFromConfig(c *config.C) (string, error) {
	caPathOrPEM := c.GetString("pki.ca", "")
	if caPathOrPEM == "" {
		return "", errors.New("no pki.ca path or PEM data provided")
	}
	return caPathOrPEM, nil
}

func loadCAFromFile(l *logrus.Logger, caPathOrPEM string) (*cert.NebulaCAPool, error) {
	var rawCA []byte
	var err error

	if strings.Contains(caPathOrPEM, "-----BEGIN") {
		rawCA = []byte(caPathOrPEM)

	} else {
		rawCA, err = ioutil.ReadFile(caPathOrPEM)
		if err != nil {
			return nil, fmt.Errorf("1. unable to read pki.ca file %s: %s", caPathOrPEM, err)
		}
	}

	CAs, err := cert.NewCAPoolFromBytes(rawCA)
	if errors.Is(err, cert.ErrExpired) {
		var expired int
		for _, cert := range CAs.CAs {
			if cert.Expired(time.Now()) {
				expired++
				l.WithField("cert", cert).Warn("expired certificate present in CA pool")
			}
		}

		if expired >= len(CAs.CAs) {
			return nil, errors.New("no valid CA certificates present")
		}
	} else if err != nil {
		return nil, fmt.Errorf("error while adding CA certificate to CA trust store: %s", err)
	}

	return CAs, nil
}

func getrawCertState(networkID uint64, relay_index byte, sqlsecret string, keysecret string, sendSignRequest bool) (*CertState, error) {
	_, key, cert, err := get_network_certs(networkID, sqlsecret)
	if sendSignRequest && err != nil {
		_, key, cert, err = generate_and_sign_nh_certs(networkID, relay_index, sqlsecret, keysecret)
		if err != nil {
			return nil, err
		}
	}
	return NewCertStateFromFiles(key, cert)
}

func ValidateVPNIP(config *config.C, ip net.IP) bool {
	cs, err := NewCertStateFromConfig(config)
	if err != nil {
		return false
	}
	tunCidr := cs.certificate.Details.Ips[0]
	return tunCidr.Contains(ip)
}

func GetNetworkID(config *config.C) (uint64, error) {
	cs, err := NewCertStateFromConfig(config)
	if err != nil {
		return 0, err
	}
	return cs.certificate.Details.NetworkID, nil
}
