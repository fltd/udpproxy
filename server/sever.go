package server

import (
	"context"
	"fmt"
	"net"
	"syscall"

	"github.com/caddyserver/caddy"
	"golang.org/x/sys/unix"
)

// ProxyServer is an implementation of the
// caddy.Server interface type
type ProxyServer struct {
	LocalAddr       string
	RemoteAddr      string
	config          *Config
	udpPacketConn   net.PacketConn
	udpClients      map[string]*udpProxy
	udpClientClosed chan string
}

// NewProxyServer returns a new proxy server
func NewProxyServer(l string, r string, c *Config) (*ProxyServer, error) {
	return &ProxyServer{
		LocalAddr:  l,
		RemoteAddr: r,
		config:     c,
		udpClients: make(map[string]*udpProxy),
	}, nil
}

// Listen is no-op TCP packets listener
func (s *ProxyServer) Listen() (net.Listener, error) {
	return nil, nil
}

// ListenPacket starts listening by creating a new Packet listener
// and returning it. It does not start accepting
// connections.
func (s *ProxyServer) ListenPacket() (net.PacketConn, error) {
	listenConfig := &net.ListenConfig{
		Control: func(network, address string, c syscall.RawConn) (err error) {
			return c.Control(func(fd uintptr) {
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
					return
				}
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
					return
				}
			})
		},
	}
	conn, err := listenConfig.ListenPacket(context.Background(), "udp", fmt.Sprintf("%s", s.LocalAddr))
	if err != nil {
		return nil, fmt.Errorf("could not create the packet listener: %v", err)
	}
	return conn, nil
}

// Serve is a no-op TCP packets handler
func (s *ProxyServer) Serve(ln net.Listener) error {
	return nil
}

// ServePacket starts serving using the provided listener.
// ServePacket blocks indefinitely, or in other
// words, until the server is stopped.
func (s *ProxyServer) ServePacket(con net.PacketConn) error {
	s.udpPacketConn = con
	s.udpClientClosed = make(chan string)

	go s.handleClosedUDPConnections()

	buf := make([]byte, 4096)
	for {
		nr, addr, err := s.udpPacketConn.ReadFrom(buf)
		if err != nil {
			s.udpPacketConn.Close()
		}

		conn, found := s.udpClients[addr.String()]
		if !found {
			remoteUDPConn, err := dialUDP("", s.RemoteAddr)
			if err != nil {
				return err
			}
			var remoteAliasUDPConns []*net.UDPConn
			if len(s.config.ReplyAddrAliases) > 0 {
				for _, a := range s.config.ReplyAddrAliases {
					remoteAliasUDPConn, err := dialUDP(remoteUDPConn.LocalAddr().String(), a)
					if err != nil {
						return err
					}
					remoteAliasUDPConns = append(remoteAliasUDPConns, remoteAliasUDPConn)
				}
			}

			conn = &udpProxy{
				lconn:     s.udpPacketConn,
				laddr:     addr,
				rconn:     remoteUDPConn,
				aconns:    remoteAliasUDPConns,
				closeChan: s.udpClientClosed,
			}

			if !caddy.Quiet {
				fmt.Println("[INFO] Mapping " + addr.String() + " <-> " + conn.rconn.LocalAddr().String())
			}
			s.udpClients[addr.String()] = conn

			// wait for data from remote server
			go conn.Wait()
		}

		// proxy data received to remote server
		_, err = conn.rconn.Write(buf[0:nr])
		if err != nil {
			return err
		}
	}
}

// handleClosedUDPConnections blocks and waits for udp closed connections and do cleanup
func (s *ProxyServer) handleClosedUDPConnections() {
	for {
		clientAddr := <-s.udpClientClosed
		conn, found := s.udpClients[clientAddr]
		if found {
			conn.Close()
			delete(s.udpClients, clientAddr)
		}
	}
}

// Stop stops s gracefully and closes its listener.
func (s *ProxyServer) Stop() error {
	return s.udpPacketConn.Close()
}

// OnStartupComplete lists the sites served by this server
// and any relevant information
func (s *ProxyServer) OnStartupComplete() {
	if !caddy.Quiet {
		fmt.Println("[INFO] Proxying from ", s.LocalAddr, " -> ", s.RemoteAddr)
		if !caddy.Quiet {
			fmt.Println("[INFO]     Accept replies from ", s.RemoteAddr, " -> ", s.LocalAddr)
		}
		if len(s.config.ReplyAddrAliases) > 0 {
			for _, a := range s.config.ReplyAddrAliases {
				if !caddy.Quiet {
					fmt.Println("[INFO]     Accept replies from ", a, " -> ", s.LocalAddr)
				}
			}
		}
	}
}

func dialUDP(l string, r string) (*net.UDPConn, error) {
	var laddr net.Addr
	if l != "" {
		resolvedLocalAddr, err := net.ResolveUDPAddr("udp", l)
		if err != nil {
			return nil, err
		}
		laddr = resolvedLocalAddr
	} else {
		laddr = nil
	}
	raddr, err := net.ResolveUDPAddr("udp", r)
	if err != nil {
		return nil, err
	}
	d := net.Dialer{
		LocalAddr: laddr,
		Control: func(network, address string, c syscall.RawConn) (err error) {
			return c.Control(func(fd uintptr) {
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1); err != nil {
					return
				}
				if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, unix.SO_REUSEPORT, 1); err != nil {
					return
				}
			})
		},
	}
	conn, err := d.Dial(raddr.Network(), raddr.String())
	if err != nil {
		return nil, err
	}
	udpConn, ok := conn.(*net.UDPConn)
	if !ok {
		return nil, fmt.Errorf("could not convert the connection to an UDP connection")
	}
	if !caddy.Quiet {
		fmt.Println("[INFO] Dial up an UDP connection " + udpConn.LocalAddr().String() + " -> " + udpConn.RemoteAddr().String())
	}
	return udpConn, nil
}
