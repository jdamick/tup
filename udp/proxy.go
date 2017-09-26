package udp

import (
	"net"
	"sync"
	"time"

	"github.com/jdamick/tup/backend"
	"github.com/jdamick/tup/config"
)

const (
	udpnetwork = "udp"
	bufferSize = 9000
)

type UDPProxy struct {
	config         *config.Config
	BackendManager *backend.Manager
	clientAddr     net.UDPAddr
	backendReaders sync.Map
}

func NewUDPProxy(config *config.Config) *UDPProxy {
	return &UDPProxy{config: config}
}

func (u *UDPProxy) Start() error {
	proxyConn, proxyAddr, err := u.createConnection(u.config.ProxyAddr)
	if err != nil {
		return err
	}

	u.clientAddr = net.UDPAddr{
		IP:   proxyAddr.IP,
		Zone: proxyAddr.Zone,
		Port: 0, // any port
	}

	go u.proxyReader(proxyConn) //, func(buf []byte) {
	//	u.BackendManager.Backend()
	//})
	return nil
}
func (u *UDPProxy) logger() config.Logger {
	return u.config.Log()
}

func (u *UDPProxy) createConnection(addr string) (*net.UDPConn, *net.UDPAddr, error) {
	udpAddr, err := net.ResolveUDPAddr(udpnetwork, addr)
	if err != nil {
		u.logger().Errorf("Error resolving udp address: %v", err)
		return nil, nil, err
	}
	conn, err := net.ListenUDP(udpAddr.Network(), udpAddr)
	if err != nil {
		u.logger().Errorf("Error listening on udp address: %v", err)
		return nil, nil, err
	}
	return conn, udpAddr, nil
}

//type ProxyWriter func(buf []byte)

type clientBackend struct {
	backendClient *net.UDPConn
	backendAddr   *net.UDPAddr
	done          chan struct{}
	lastUsed      time.Time
}

func (c *clientBackend) Close() {
	if c.backendClient != nil {
		c.backendClient.Close()
	}
}

func (u *UDPProxy) proxyReader(conn *net.UDPConn /*, writer ProxyWriter*/) {
	buf := make([]byte, bufferSize)
	for {
		cnt, srcAddr, err := conn.ReadFromUDP(buf)

		if err != nil {
			u.logger().Errorf("Error reading: %v", err)
			continue // unless we need to quit
		}

		u.logger().Debugf("Got connection from: %v", srcAddr)

		key := makeKey(srcAddr)

		var backendConnInfo interface{}
		var loaded bool
		if backendConnInfo, loaded = u.backendReaders.Load(key); !loaded {
			u.logger().Debugf("Creating backend connection")
			// client to backend
			client, err := net.ListenUDP(udpnetwork, &u.clientAddr)
			if err != nil {
				u.logger().Errorf("Setting up client: %v", err)
				return
			}

			backend := u.BackendManager.Backend()
			udpAddr, err := net.ResolveUDPAddr(udpnetwork, backend.Addr)
			u.logger().Debugf("Resolved backend to: %v", udpAddr)
			if err != nil {
				u.logger().Errorf("Error resolving udp address: %v", err)
				return
			}

			backendConnInfo = &clientBackend{backendClient: client, backendAddr: udpAddr, lastUsed: time.Now()}
			var existingBackendConnInfo interface{}
			if existingBackendConnInfo, loaded = u.backendReaders.LoadOrStore(key, backendConnInfo); loaded {
				backendConnInfo.(*clientBackend).Close()
				backendConnInfo = existingBackendConnInfo
				bk := existingBackendConnInfo.(*clientBackend)
				bk.lastUsed = time.Now()
				u.logger().Debugf("Updated backend timestamp: %v", bk.lastUsed)
			}
		}

		// send to backend
		bckInfo := backendConnInfo.(*clientBackend)
		bckInfo.backendClient.WriteTo(buf[:cnt], bckInfo.backendAddr)

		if !loaded {
			go u.proxyReturn(conn, srcAddr, bckInfo.backendClient)
		}
	}
}

func makeKey(proxyClientSrcAddr *net.UDPAddr) string {
	return proxyClientSrcAddr.String()
}

func (u *UDPProxy) proxyReturn(proxyClient *net.UDPConn, proxyClientSrcAddr *net.UDPAddr, backendClient *net.UDPConn) {
	buf := make([]byte, bufferSize)

	defer u.logger().Debugf("Closing proxy return pump")

	for i := 0; ; i++ {
		if i > 0 {
			// be aggressive on cleanup, second time around if there is no data then just close up shop.
			backendClient.SetDeadline(time.Now().Add(time.Millisecond * 100))
		}
		cnt, _, err := backendClient.ReadFromUDP(buf)
		if err != nil {
			// remove myself
			u.backendReaders.Delete(makeKey(proxyClientSrcAddr))
			backendClient.Close()
			if i == 0 {
				u.logger().Errorf("Error reading udp from backend: %v", err)
			}
			return
		}
		u.logger().Debugf("Read from backend: %v bytes and sending back to client: %v", cnt, proxyClientSrcAddr)
		wcnt, err := proxyClient.WriteTo(buf[:cnt], proxyClientSrcAddr)
		if err != nil {
			u.logger().Errorf("Write error reading udp from backend: %v", err)
		}
		if wcnt != cnt {
			u.logger().Errorf("Write error, should have written: %v but wrote: %v", cnt, wcnt)
		}
	}
}
