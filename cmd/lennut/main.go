package main

import (
	"flag"
	"log"
	"os"
	"os/signal"

	"github.com/banks/lennut"
)

func main() {
	var server bool
	var s lennut.Server
	var c lennut.Client

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
