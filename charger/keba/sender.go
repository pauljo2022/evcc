package keba

import (
	"io"
	"net"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Sender is a KEBA UDP sender
type Sender struct {
	log  api.Logger
	addr string
	conn *net.UDPConn
}

// NewSender creates KEBA UDP sender
func NewSender(log api.Logger, addr string) (*Sender, error) {
	addr = util.DefaultPort(addr, Port)
	raddr, err := net.ResolveUDPAddr("udp", addr)

	var conn *net.UDPConn
	if err == nil {
		conn, err = net.DialUDP("udp", nil, raddr)
	}

	c := &Sender{
		log:  log,
		addr: addr,
		conn: conn,
	}

	return c, err
}

// Send msg to receiver
func (c *Sender) Send(msg string) error {
	c.log.Tracef("send to %s %v", c.addr, msg)
	_, err := io.Copy(c.conn, strings.NewReader(msg))
	return err
}
