package main

import (
	"bufio"
	"fmt"
	"log"
	"net"
)

func startC() {
	tunnelPort := cfg.C.PortTunnel
	if tunnelPort == 0 {
		log.Fatal("[C] Q1 tunnel port invalid, it is", tunnelPort)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", tunnelPort))
	if err != nil {
		log.Fatal("[C] Q2 failed to listen port", tunnelPort, err)
	}
	log.Printf("[C] service listening on :%d", tunnelPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("[C] accept connection failed", err)
			continue
		}
		go handleClient(conn)
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	log.Println("[C] new connection from", conn.RemoteAddr())
	conn.Write([]byte("Hello from C! You reached me via A->B->C.\n"))

	// 回显
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		msg := scanner.Text()
		log.Printf("[C] received: %s", msg)
		conn.Write([]byte("Echo: " + msg + "\n"))
	}
}
