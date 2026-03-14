package main

import (
	"io"
	"log"
	"net"
	"time"
)

type connDirection int

const (
	dirNone connDirection = iota
	dirUser
	dirTunnel
	dirA
	dirC
	dirForward
)

func sleep(ms int) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

func Relay(a, b net.Conn) {
	log.Printf("Relay: %s <-> %s", a.RemoteAddr(), b.RemoteAddr())
	go func() {
		io.Copy(a, b)
		a.Close()
		b.Close()
	}()
	go func() {
		io.Copy(b, a)
		a.Close()
		b.Close()
	}()
}
