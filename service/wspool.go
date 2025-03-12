package service

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
)

type UDPPacket struct {
	content [4096]byte
	n       int
}

type WSConnection struct {
	conn      *websocket.Conn
	WSURL     string // Empty when on server side, disable autorestart
	closing   bool
	redialM   sync.RWMutex
	redialC   *sync.Cond
	heartbeat chan bool
}

type WSConnectionPool struct {
	wsSendChannel  chan *UDPPacket
	udpSendChannel chan *UDPPacket
	udpListenConn  *net.UDPConn
	udpTargetAddr  net.Addr
	closing        bool
	bufferPool     sync.Pool
	RecvUDPPacket  atomic.Uint64
	SentUDPPacket  atomic.Uint64
	RecvWSPacket   atomic.Uint64
	SentWSPacket   atomic.Uint64
}

func (conn *WSConnection) CanDial() bool {
	return conn.WSURL != ""
}

func (conn *WSConnection) Dial() {
	for !conn.closing {
		conn.redialM.Lock()
		conn.redialC.Wait()
		var err error
		if conn.CanDial() {
			if conn.conn != nil {
				conn.conn.Close()
			}
			conn.conn, _, err = websocket.DefaultDialer.Dial(conn.WSURL, nil)
			for err != nil {
				log.Printf("Failed to re-dial, retry in 5s: %v", err)
				conn.conn, _, err = websocket.DefaultDialer.Dial(conn.WSURL, nil)
				time.Sleep(5 * time.Second)
			}
		}
		conn.redialM.Unlock()
	}
	conn.conn.Close()
}

func (c *WSConnectionPool) Init(udpListenAddr *net.UDPAddr, udpTargetAddr net.Addr) error {
	var err error
	c.udpListenConn, err = net.ListenUDP("udp", udpListenAddr)
	if err != nil {
		return err
	}
	c.bufferPool = sync.Pool{
		New: func() interface{} {
			return &UDPPacket{}
		},
	}

	c.udpTargetAddr = udpTargetAddr

	c.wsSendChannel = make(chan *UDPPacket, 1024)
	c.udpSendChannel = make(chan *UDPPacket, 1024)
	// c.wsSendChannel = make(chan *UDPPacket)
	// c.udpSendChannel = make(chan *UDPPacket)

	return nil
}

func (c *WSConnectionPool) NewConnection(conn *WSConnection) error {
	var err error
	if conn.conn == nil {
		conn.conn, _, err = websocket.DefaultDialer.Dial(conn.WSURL, nil)
		if err != nil {
			return err
		}
		go conn.Dial()
	}
	conn.redialC = sync.NewCond(&conn.redialM)
	conn.heartbeat = make(chan bool, 3)
	go c.handleConnectionSent(conn)
	go c.handleConnectionListen(conn)
	go c.handleConnectionHeartbeat(conn)
	return nil
}

// Return: can resume
func (conn *WSConnection) wsFail() bool {
	if conn.CanDial() {
		conn.redialC.Signal()
		return true
	} else {
		conn.closing = true
		return false
	}
}

func (c *WSConnectionPool) handleConnectionHeartbeat(conn *WSConnection) {
	for !conn.closing {
		time.Sleep(2 * time.Second)
		conn.heartbeat <- true
	}
	close(conn.heartbeat)
}

func (c *WSConnectionPool) handleConnectionListen(conn *WSConnection) {
	for !conn.closing {
		conn.redialM.RLock()
		messageType, r, err := conn.conn.NextReader()
		if err != nil {
			conn.redialM.RUnlock()
			log.Printf("Failed to read message from WebSocket: %v", err)
			if conn.wsFail() {
				continue
			} else {
				return
			}
		}
		if messageType != websocket.BinaryMessage {
			conn.redialM.RUnlock()
			continue
		}
		packet := c.bufferPool.Get().(*UDPPacket)
		n, err := r.Read(packet.content[:])
		conn.redialM.RUnlock()
		// log.Printf("[DEBUG %s] Packet received from Websocket", time.Now().String())
		if err != nil {
			log.Printf("Failed to read message from WebSocket: %v", err)
			c.bufferPool.Put(packet)
			if conn.wsFail() {
				continue
			} else {
				return
			}
		}
		packet.n = n
		c.udpSendChannel <- packet
		c.RecvWSPacket.Add(1)
	}
}

func (c *WSConnectionPool) handleConnectionSent(conn *WSConnection) {
	for !conn.closing {
		select {

		case packet, ok := <-c.wsSendChannel:
			if !ok {
				break
			}
			if conn.closing {
				c.wsSendChannel <- packet
				return
			}
			conn.redialM.RLock()
			err := conn.conn.WriteMessage(websocket.BinaryMessage, packet.content[:packet.n])
			conn.redialM.RUnlock()
			// log.Printf("[DEBUG %s] Packet sent to Websocket", time.Now().String())
			if err != nil {
				log.Printf("Failed to send message to WebSocket: %v", err)
				c.wsSendChannel <- packet
				if conn.wsFail() {
					continue
				} else {
					return
				}
			} else {
				c.SentWSPacket.Add(1)
				c.bufferPool.Put(packet)
			}
		case _, ok := <-conn.heartbeat:
			if !ok {
				break
			}
			if conn.closing {
				return
			}
			conn.redialM.RLock()
			err := conn.conn.WriteMessage(websocket.PingMessage, nil)
			conn.redialM.RUnlock()
			if err != nil {
				log.Printf("Failed to send message to WebSocket: %v", err)
				if conn.wsFail() {
					continue
				} else {
					return
				}
			}
		}
	}
}

func (c *WSConnectionPool) webSocketToUDP() {
	for {
		packet, ok := <-c.udpSendChannel
		if !ok {
			break
		}
		_, err := c.udpListenConn.WriteTo(packet.content[:packet.n], c.udpTargetAddr)
		// log.Printf("[DEBUG %s] Packet sent to UDP", time.Now().String())
		if err != nil {
			log.Printf("Failed to send message to UDP: %v", err)
		}
		c.SentUDPPacket.Add(1)
		c.bufferPool.Put(packet)
	}
}

func (c *WSConnectionPool) udpToWebSocket() {
	for !c.closing {
		packet := c.bufferPool.Get().(*UDPPacket)
		n, _, err := c.udpListenConn.ReadFromUDP(packet.content[:])
		// log.Printf("[DEBUG %s] Packet received from UDP", time.Now().String())
		packet.n = n
		c.RecvUDPPacket.Add(1)
		if err != nil {
			log.Printf("Failed to read message from UDP: %v", err)
			c.bufferPool.Put(packet)
			continue
		}
		c.wsSendChannel <- packet
	}
}

func (c *WSConnectionPool) Close() {
	c.closing = true
	close(c.wsSendChannel)
	close(c.udpSendChannel)
	c.udpListenConn.Close()
}
