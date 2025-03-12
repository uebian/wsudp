package service

import (
	"net"

	"github.com/uebian/wsudp/config"
)

type Client struct {
	wspool  *WSConnectionPool
	cfg     *config.Config
	closing bool
}

func NewClient(cfg *config.Config) *Client {
	client := &Client{
		wspool:  &WSConnectionPool{Collector: NewWSUDPCollector()},
		closing: false,
		cfg:     cfg,
	}

	return client
}

func (c *Client) Init() error {
	udpListenAddr, err := net.ResolveUDPAddr("udp", c.cfg.Client.UDPListenAddr)
	if err != nil {
		return err
	}

	udpTargetAddr, err := net.ResolveUDPAddr("udp", c.cfg.Client.UDPTargetAddr)
	if err != nil {
		return err
	}

	err = c.wspool.Init(udpListenAddr, udpTargetAddr)
	if err != nil {
		return err
	}

	for i := 0; i < c.cfg.Client.NMux; i++ {
		err = c.wspool.NewConnection(&WSConnection{WSURL: c.cfg.Client.WSURL})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Close() error {
	c.closing = true
	c.wspool.Close()
	return nil
}

func (c *Client) ListenAndServe() {
	go c.wspool.udpToWebSocket()
	c.wspool.webSocketToUDP()
}
