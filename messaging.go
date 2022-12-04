package nebula

import (
	"container/ring"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	messages "messages"
	nh_util "nh_util"

	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula/header"
	"github.com/slackhq/nebula/iputil"
)

const MAX_MESSAGES_PER_IP = 10
const MAX_RETRIES = 3
const MH_HEADER_LEN = 16
const MH_ACK_IN_MESSAGE = 0x1
const MAX_PACKET_SIZE = 9000
const MAX_NUMBER_OF_EVENTS = 8

var ErrMHHeaderTooShort = errors.New("MH header is too short")

type MessageManager struct {
	acked     [MAX_MESSAGES_PER_IP]chan bool
	attemps   [MAX_RETRIES]uint8 // Number of attempts
	vpnIP     iputil.VpnIp
	Seqnum    uint32
	nextindex uint8
	lock      *sync.RWMutex
	pending   [MAX_MESSAGES_PER_IP]bool
	recvdmsg  [MAX_MESSAGES_PER_IP][]byte
}

// Messaing header
type MH struct {
	Version   uint8
	flags     uint8
	reserved2 uint16
	Seqnum    uint32
	Acknum    uint32
	checksum  uint32
}

type Messaging struct {
	messages  map[iputil.VpnIp]*MessageManager
	l         *logrus.Logger
	f         *Interface
	EventRing *ring.Ring
}

func NewMessageManager() *MessageManager {
	m := &MessageManager{}
	m.lock = &sync.RWMutex{}
	for i := 0; i < MAX_MESSAGES_PER_IP; i++ {
		m.acked[i] = make(chan bool)
	}
	return m
}

func NewMH() *MH {
	return &MH{}
}

func MHEncode(b []byte, version uint8, flags uint8, seqnum uint32, acknum uint32, checksum uint32) []byte {
	b[0] = byte(version)
	b[1] = byte(flags)
	binary.BigEndian.PutUint32(b[4:8], seqnum)
	binary.BigEndian.PutUint32(b[8:12], acknum)
	binary.BigEndian.PutUint32(b[12:16], checksum)
	return b
}

func (mh *MH) MHParse(b []byte) error {
	if len(b) < MH_HEADER_LEN {
		return ErrMHHeaderTooShort
	}
	mh.Version = b[0]
	mh.flags = b[1]
	mh.Seqnum = binary.BigEndian.Uint32(b[4:8])
	mh.Acknum = binary.BigEndian.Uint32(b[8:12])
	mh.checksum = binary.BigEndian.Uint32(b[12:16])

	return nil
}

func NewMessaging(ll *logrus.Logger, ifce *Interface) *Messaging {
	return &Messaging{
		messages:  make(map[iputil.VpnIp]*MessageManager),
		l:         ll,
		f:         ifce,
		EventRing: ring.New(MAX_NUMBER_OF_EVENTS),
	}
}

func (m *Messaging) Run(vpnIp iputil.VpnIp, seqnum uint32) bool {
	clockSource := time.NewTicker(8 * time.Second)
	defer clockSource.Stop()
	status := false
	select {
	case status = <-m.messages[vpnIp].acked[seqnum]:
		break
	case _ = <-clockSource.C:
		break
	}
	return status
}

func (m *Messaging) sendMessage(vpnIp iputil.VpnIp, networkID uint64, packet []byte, subtype header.MessageSubType, seqnum uint32) (string, error) {
	l := len(packet)
	packet1 := make([]byte, l+MH_HEADER_LEN)
	out := make([]byte, mtu)
	nb := make([]byte, 12, 12)

	if m.messages[vpnIp] == nil {
		msg := NewMessageManager()
		m.messages[vpnIp] = msg
	}
	if seqnum >= MAX_MESSAGES_PER_IP {
		return "", fmt.Errorf("Some bigger sequence number")
	}

	m.messages[vpnIp].lock.Lock()
	defer m.messages[vpnIp].lock.Unlock()

	var flags uint8
	flags = 0
	if subtype == header.NonTunMessageACK {
		flags = flags | MH_ACK_IN_MESSAGE
	}
	checksum := nh_util.NH_checksum(packet)
	packet1 = MHEncode(packet1, 0, flags, m.messages[vpnIp].Seqnum, seqnum, checksum)

	copy(packet1[MH_HEADER_LEN:MH_HEADER_LEN+l], packet)
	len1 := len(packet1)

	hostinfo := m.f.getOrHandshake(vpnIp, networkID, true)
	if hostinfo == nil {
		return "", fmt.Errorf("Can't reach the host")
	}
	ci := hostinfo.ConnectionState

	if ci.ready == false {
		return "", fmt.Errorf("Host not ready")
	}

	// Send Wait for the ACK
	var status bool
	var i int
	m.messages[vpnIp].pending[m.messages[vpnIp].Seqnum] = true
	for i = 0; i < 3; i++ {
		m.f.sendNoMetrics(header.NonTunMessage, subtype, ci, hostinfo, hostinfo.remote, packet1[:len1], nb, out, 0)
		if subtype == header.NonTunMessageACK {
			// No retransmissions for ACK messages
			return "", fmt.Errorf("Not sure if this ack is sent")
		}
		status = m.Run(vpnIp, m.messages[vpnIp].Seqnum)
		if status {
			break
		}
	}

	if i == 3 {
		return "", fmt.Errorf("Message not sent. Try again")
	}
	m.messages[vpnIp].pending[m.messages[vpnIp].Seqnum] = false
	retout := m.messages[vpnIp].recvdmsg[m.messages[vpnIp].Seqnum]

	for i := 0; i < MAX_MESSAGES_PER_IP; i++ {
		if !m.messages[vpnIp].pending[i] {
			// Got the next sequence number
			m.messages[vpnIp].Seqnum = (m.messages[vpnIp].Seqnum + 1) % MAX_MESSAGES_PER_IP
			break
		}
	}
	return string(retout), nil
}

func (m *Messaging) recvMessage(vpnIp iputil.VpnIp, msg []byte) {
	mh := NewMH()

	mh.MHParse(msg)
	inmsg := msg[MH_HEADER_LEN:]
	checksum := nh_util.NH_checksum(inmsg)
	m.l.Error("Received message...", string(inmsg))

	if mh.checksum != checksum {
		fmt.Println("Checksum mismatch ", mh.checksum, checksum)
		fmt.Println("mh.Seqnum ", mh.Seqnum)
		m.l.Error("Messaging: recevMessage, Checksum mismatch ", mh.checksum, checksum)
		return
	}
	if mh.Acknum >= MAX_MESSAGES_PER_IP {
		// Some index greater than the max. Suspicious
		return
	}
	if mh.flags&MH_ACK_IN_MESSAGE > 0 {
		m.messages[vpnIp].acked[mh.Acknum] <- true
		m.messages[vpnIp].recvdmsg[mh.Acknum] = inmsg
	} else {
		ret := messages.ProcessMessage(inmsg, m.l, m.EventRing)
		m.l.Error("Sending the message ", ret)
		go m.sendMessage(vpnIp, m.f.networkID, []byte(ret), header.NonTunMessageACK, mh.Seqnum)
	}
}

func (m *Messaging) GetNextEvent() string {
	var ev *messages.Event
	ev = nil

	m.EventRing.Do(func(p interface{}) {
		if p != nil && ev == nil {
			event := p.(*messages.Event)
			if event.Active {
				ev = event
				ev.Active = false
			}
		}
	})
	if ev != nil {
		jsonData, err := json.Marshal(ev)
		if err != nil {
			m.l.Error("Error while marshalling Event data..", err.Error())
			return ""
		}
		return string(jsonData)
	}
	return ""
}
