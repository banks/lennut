package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync/atomic"
	"time"
)

type Client struct {
	DialAddr string
	ProxyTo  string
	stopped  int32
	stopCh   chan struct{}
}

// Run runs the client proxy. It will dial the given server and destination
// endpoint and if successful will proxy bytes between the two.
func (c *Client) Run() error {
	c.stopCh = make(chan struct{})
	waitS := float64(1)

	log.Printf("INFO starting client connecting to %s, proxying to %s",
		c.DialAddr, c.ProxyTo)

	onErr := func(where string, err error) {
		if atomic.LoadInt32(&c.stopped) == 1 {
			return
		}
		wait := time.Duration(waitS) * time.Second
		log.Printf("ERR %s, retry in %s: %s", where, wait, err)
		select {
		case <-time.After(wait):
		case <-c.stopCh:
		}
		waitS *= 1.5
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		<-c.stopCh
		cancel()
	}()

	var d net.Dialer

	for {
		if atomic.LoadInt32(&c.stopped) == 1 {
			return nil
		}

		// Attempt to dial server and wait for data before opening another
		src, err := d.DialContext(ctx, "tcp", c.DialAddr)
		if err != nil {
			onErr("dialing lennut server", err)
			continue
		}

		// Close the conn if we stop so the read below unblocks
		go func() {
			<-c.stopCh
			src.Close()
		}()

		log.Println("INFO established conn to server, waiting for incoming")

		// Wait until the end client connects to server and starts sending data.
		// lennut server always sends a 0xff byte on the conn when a connection
		// comes in.
		var b [HeaderLen]byte
		_, err = src.Read(b[:])
		if err != nil {
			src.Close()
			onErr("reading initial header from server", err)
			continue
		}
		if string(b[:]) != MagicBytes {
			src.Close()
			err := fmt.Errorf("expected initial header %q from server, got %q", MagicBytes, string(b[:]))
			onErr("protocol error: %s", err)
			continue
		}

		// Now there is a connection inbound from the end client, dial the backend
		dst, err := d.DialContext(ctx, "tcp", c.ProxyTo)
		if err != nil {
			src.Close()
			onErr("dialing backend server", err)
			continue
		}

		log.Println("INFO got header, proxying to backend")

		// Handle proxying in background and loop to open a new conn for next client
		go proxyBytes(c.stopped, c.stopCh, src, dst)

		// Success, reset wait time
		waitS = 1
	}
	return nil
}

// Close terminates the client
func (c *Client) Close() error {
	old := atomic.SwapInt32(&c.stopped, 1)
	if old == 0 {
		close(c.stopCh)
	}
	return nil
}
