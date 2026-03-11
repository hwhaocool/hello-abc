package main

import (
	"io"
	"log"
	"net"
)

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
