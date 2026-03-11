package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
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

func handleClient(tunnelConn net.Conn) {
	defer tunnelConn.Close()
	log.Printf("[C] new tunnel connection from %s\n", tunnelConn.RemoteAddr())

	for {
		// 连接转发目标
		forwardConn, err := connectForward()
		if err != nil {
			log.Println("[C] Q5 failed to connect forward, retrying in 3s", err)
			time.Sleep(3 * time.Second)
			continue
		}

		log.Printf("[C] connected to forward, starting relay tunnel<->forward\n")

		// 中继数据
		var wg sync.WaitGroup
		forwardCloseOnce := sync.Once{}
		wg.Add(2)

		go func() {
			defer wg.Done()
			defer func() {
				forwardCloseOnce.Do(func() {
					log.Printf("[C] closing forward connection: %s\n", forwardConn.RemoteAddr())
					forwardConn.Close()
				})
			}()
			relayOneWayC(tunnelConn, forwardConn)
		}()

		go func() {
			defer wg.Done()
			defer func() {
				forwardCloseOnce.Do(func() {
					log.Printf("[C] closing forward connection: %s\n", forwardConn.RemoteAddr())
					forwardConn.Close()
				})
			}()
			relayOneWayC(forwardConn, tunnelConn)
		}()

		// 等待两个方向都完成
		wg.Wait()

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
	log.Printf("[C] connected to forward from %s\n", conn.RemoteAddr())
	return conn, nil
}

func relayOneWayC(dst, src net.Conn) {
	defer func() {
		log.Printf("[C] relay direction: %s -> %s completed\n", src.RemoteAddr(), dst.RemoteAddr())
	}()
	io.Copy(dst, src)
}
