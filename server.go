package main

import (
	"log"
	"net"
	"sync/atomic"
	"time"
)

// Server is a server
type Server struct {
	ListenInboundAddr string
	ListenClientAddr  string
	waiting           chan net.Conn
	stopped           int32
	stopCh            chan struct{}
}

// Run runs the proxy server
func (s *Server) Run() error {
	s.waiting = make(chan net.Conn, 128)
	s.stopCh = make(chan struct{})

	log.Printf("INFO starting server listening for clients on %s, proxying conns from %s",
		s.ListenClientAddr, s.ListenInboundAddr)

	l, err := net.Listen("tcp", s.ListenInboundAddr)
	if err != nil {
		return err
	}
	defer l.Close()

	lc, err := net.Listen("tcp", s.ListenClientAddr)
	if err != nil {
		return err
	}
	defer lc.Close()

	go s.listenInbound(l)
	go s.listenClient(lc)

	<-s.stopCh
	return nil
}

func (s *Server) listenInbound(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if atomic.LoadInt32(&s.stopped) == 1 {
				return nil
			}
			log.Printf("ERR accepting inbound conn: %s", err)
		}

		// Find a waiting connection
		select {
		case dst := <-s.waiting:
			go s.proxyConn(conn, dst)
		default:
			// No waiters, just close it
			conn.Close()
			log.Printf("WARN closing conn from %s no, waiting clients", conn.RemoteAddr().String())
		}
	}
}

func (s *Server) proxyConn(src, dst net.Conn) {
	defer dst.Close()
	defer src.Close()

	// Got a conn, send the magic handshake bytes to let it know a new
	// connection is being established.
	err := dst.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if err != nil {
		log.Printf("ERR failed to set deadline: %s", err)
		return
	}
	n, err := dst.Write([]byte(MagicBytes))
	if err != nil || n != HeaderLen {
		log.Printf("ERR failed to write header: %s", err)
		return
	}
	// Reset write deadline so it doesn't timeout proxy connection
	err = dst.SetWriteDeadline(time.Time{})
	if err != nil {
		log.Printf("ERR failed to set reset deadline: %s", err)
		return
	}

	log.Printf("INFO accepted conn from %s connecting with client %s",
		src.RemoteAddr().String(), dst.RemoteAddr().String())

	// OK, proxy the rest directly
	proxyBytes(s.stopped, s.stopCh, src, dst)
}

func (s *Server) listenClient(l net.Listener) error {
	for {
		conn, err := l.Accept()
		if err != nil {
			if atomic.LoadInt32(&s.stopped) == 1 {
				return nil
			}
			log.Printf("ERR accepting client conn: %s", err)
		}

		// Add as a waiting connection
		select {
		case s.waiting <- conn:
		default:
			// Too many waiters, just close
			conn.Close()
		}
	}
}

// Close stops the server
func (s *Server) Close() error {
	old := atomic.SwapInt32(&s.stopped, 1)
	if old == 0 {
		close(s.stopCh)
	}
	return nil
}
