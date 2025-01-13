package service

import (
	"fmt"
	"log"
	"net"
	"net/http"

	//_ "net/http/pprof"

	"github.com/gorilla/websocket"
	"github.com/uebian/wsudp/config"
)

type Server struct {
	wspool *WSConnectionPool
	cfg    *config.Config
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func NewServer(cfg *config.Config) *Server {
	svc := &Server{wspool: &WSConnectionPool{}, cfg: cfg}
	return svc
}

func (s *Server) Init() error {
	udpListenAddr, err := net.ResolveUDPAddr("udp", s.cfg.Server.UDPListenAddr)
	if err != nil {
		return err
	}

	udpTargetAddr, err := net.ResolveUDPAddr("udp", s.cfg.Server.UDPTargetAddr)
	if err != nil {
		return err
	}
	err = s.wspool.Init(udpListenAddr, udpTargetAddr)
	return err
}

func (s *Server) Close() error {
	s.wspool.Close()
	return nil
}

func (s *Server) ListenAndServe() {
	go s.wspool.udpToWebSocket()
	go s.wspool.webSocketToUDP()

	http.HandleFunc(s.cfg.Server.ListenPath, s.handleWebSocket)
	http.HandleFunc("/stats", s.handleStats)

	err := http.ListenAndServe(s.cfg.Server.WSListenAddr, nil)
	if err != nil {
		log.Fatalf("Failed to start Websocket Listener: %v", err)
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Failed to upgrade websocket: %v", err)
		return
	}
	s.wspool.NewConnection(&WSConnection{conn: conn})

}

func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(fmt.Sprintf("SentWSPacket: %d, RecvWSPacket: %d, SentUDPPacket: %d, RecvUDPPacket: %d",
		s.wspool.SentWSPacket.Load(),
		s.wspool.RecvWSPacket.Load(),
		s.wspool.SentUDPPacket.Load(),
		s.wspool.RecvUDPPacket.Load())))

}
