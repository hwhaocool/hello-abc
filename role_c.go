package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func startC() {
	checkConfigC()

	tunnelPort := cfg.C.PortTunnel
	if tunnelPort == 0 {
		log.Fatal("[C] Q1 tunnel port invalid, it is", tunnelPort)
	}
	listener, err := net.Listen("tcp", fmt.Sprintf(":%d", tunnelPort))
	if err != nil {
		log.Fatal("[C] Q2 failed to listen port", tunnelPort, err)
	}
	log.Printf("[C] service listening on :%d\n", tunnelPort)

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("[C] accept connection failed", err)
			continue
		}
		go handleClient(conn)
	}
}

func checkConfigC() {
	if cfg.C.PortTunnel == 0 {
		log.Fatalln("[C] Q3, port_tunnel not set")
	}

	if cfg.C.PortForward == 0 {
		log.Fatalln("[C] Q4, port_forward not set")
	}
}

func handleClient(conn net.Conn) {
	defer conn.Close()
	log.Println("[C] new connection from", conn.RemoteAddr())

	for {
		// 连接转发目标
		forwardConn, err := connectForward()
		if err != nil {
			log.Println("[C] Q5 failed to connect forward, retrying in 3s", err)
			time.Sleep(3 * time.Second)
			continue
		}

		log.Println("[C] connected to forward, starting relay")

		// 中继数据
		doneRelay := make(chan struct{})
		go func() {
			defer close(doneRelay)
			if _, err := relayOneWayC(conn, forwardConn); err != nil {
				log.Printf("[C] relay conn->forward error: %v", err)
			}
		}()
		go func() {
			defer close(doneRelay)
			if _, err := relayOneWayC(forwardConn, conn); err != nil {
				log.Printf("[C] relay forward->conn error: %v", err)
			}
		}()

		// 等待任一方向断开
		<-doneRelay
		log.Println("[C] forward connection lost, reconnecting in 3s...")
		time.Sleep(3 * time.Second)
	}
}

func connectForward() (net.Conn, error) {
	log.Println("[C] connecting to forward")
	conn, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.C.PortForward))
	if err != nil {
		return nil, err
	}
	log.Println("[C] connected to forward")
	return conn, nil
}

func relayOneWayC(dst, src net.Conn) (int64, error) {
	defer dst.Close()
	defer src.Close()
	return io.Copy(dst, src)
}
