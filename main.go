package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
)

const (
	// MagicBytes is the 8 byte header we expect to see at the start of a connection
	// from server to client to indicate there is an end-client connection ready to
	// start proxying. It's a string just because it's a constant and it's exactly 8
	// bytes because that seemed nice.
	MagicBytes = "lennet01"

	// HeaderLen is the length of the magic header the server sends the client
	// when a new connection is established.
	HeaderLen = 8
)

func main() {
	var server bool
	var s Server
	var c Client

	flag.BoolVar(&server, "server", false, "whether to run as a server")
	flag.StringVar(&s.ListenInboundAddr, "bind-proxy", "0.0.0.0:3000",
		"server address:port to listen for inbound connections to proxy")
	flag.StringVar(&s.ListenClientAddr, "bind-client", "0.0.0.0:3001",
		"server address:port to listen for inbound clients to proxy connections to")
	flag.StringVar(&c.DialAddr, "server-addr", "localhost:3001",
		"client address:port to connect to server on")
	flag.StringVar(&c.ProxyTo, "proxy-to", "localhost:8080",
		"client address:port to proxy connections to")

	flag.Parse()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt, os.Kill)

	go func() {
		sig := <-sigs
		log.Printf("INFO caught signal %s, exiting", sig)
		if server {
			s.Close()
		} else {
			c.Close()
		}
	}()

	var err error
	if server {
		err = s.Run()
	} else {
		err = c.Run()
	}
	if err != nil {
		log.Fatal(err)
	}
}
