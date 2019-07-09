package server

import (
	"fmt"
	"net"

	"github.com/caddyserver/caddy"
)

// udpProxy resembles a UDP proxy connection and pipe data between local and remote.
type udpProxy struct {
	lconn     net.PacketConn
	laddr     net.Addr     // Address of the client
	rconn     *net.UDPConn // UDP connection to remote server
	aconns    []*net.UDPConn
	closeChan chan string
}

// Wait reads packets from remote server and forwards it on to the client connection
func (p *udpProxy) Wait() {
	for _, aconn := range p.aconns {
		go p.WaitSingle(aconn)
	}
	p.WaitSingle(p.rconn)
}

func (p *udpProxy) WaitSingle(conn *net.UDPConn) {
	if !caddy.Quiet {
		fmt.Println("[INFO] Waiting reply on " + conn.RemoteAddr().String() + " -> " + conn.LocalAddr().String())
	}
	buf := make([]byte, 32*1024) // THIS SHOULD BE CONFIGURABLE
	for {
		// Read from server
		n, err := conn.Read(buf)
		if !caddy.Quiet {
			fmt.Println("[INFO] Got reply from " + conn.RemoteAddr().String() + " -> " + conn.LocalAddr().String())
		}
		if err != nil {
			p.closeChan <- p.laddr.String()
			return
		}
		if !caddy.Quiet {
			fmt.Println("[INFO] Writing back the reply to " + p.laddr.String())
		}
		// Relay data from remote back to client
		_, err = p.lconn.WriteTo(buf[0:n], p.laddr)
		if err != nil {
			p.closeChan <- p.laddr.String()
			return
		}
	}
}

func (p *udpProxy) Close() {
	closeConn(p.rconn)
	for _, aconn := range p.aconns {
		closeConn(aconn)
	}
}

func closeConn(conn *net.UDPConn) {
	if !caddy.Quiet {
		fmt.Println("[INFO] Closing the connection " + conn.LocalAddr().String() + " -> " + conn.RemoteAddr().String())
	}
	conn.Close()
}
