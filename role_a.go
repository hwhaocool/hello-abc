package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

func startA() {
	tunnelPort := cfg.A.PortTunnel
	if tunnelPort == 0 {
		log.Fatal("[A] Q1 tunnel port invalid, it is", tunnelPort)
	}
	// 1. 监听来自 B 的隧道连接
	tunnelLn, err := net.Listen("tcp", fmt.Sprintf(":%d", tunnelPort))
	if err != nil {
		log.Fatal("[A] Q2 listen tunnel port failed", tunnelPort, err)
	}
	defer tunnelLn.Close()
	log.Printf("[A] waiting for B connection at port %d\n", tunnelPort)

	// 2. 监听本地用户
	serverPort := cfg.A.PortServer
	if serverPort == 0 {
		log.Fatal("[A] Q4 server port invalid, it is", serverPort)
	}
	userLn, err := net.Listen("tcp", fmt.Sprintf(":%d", serverPort))
	if err != nil {
		log.Fatal("[A] Q5 listen server port failed", serverPort, err)
	}
	defer userLn.Close()
	log.Printf("[A] waiting for user connection at port %d\n", serverPort)

	// 用于保护连接的互斥锁
	var tunnelMutex sync.Mutex
	var userMutex sync.Mutex
	var tunnelConn net.Conn
	var userConn net.Conn

	// 持续接受 B 的隧道连接
	go func() {
		for {
			conn, err := tunnelLn.Accept()
			if err != nil {
				log.Println("[A] Q3 accept B connection failed", err)
				continue
			}

			log.Println("[A] new B connection, replacing old closing")

			tunnelMutex.Lock()
			oldConn := tunnelConn
			tunnelConn = conn
			tunnelMutex.Unlock()

			if oldConn != nil {
				log.Printf("[A] closing old B connection: %s\n", oldConn.RemoteAddr())
				oldConn.Close()
			}

			log.Printf("[A] B connected to tunnel from %s\n", conn.RemoteAddr())
		}
	}()

	// 持续接受用户连接
	for {
		conn, err := userLn.Accept()
		if err != nil {
			log.Println("[A] Q6 accept user connection failed", err)
			continue
		}

		log.Println("[A] new user connection, replacing old and closing")

		userMutex.Lock()
		oldConn := userConn
		userConn = conn
		userMutex.Unlock()

		if oldConn != nil {
			log.Printf("[A] closing old user connection: %s\n", oldConn.RemoteAddr())
			oldConn.Close()
		}

		log.Printf("[A] user connected from %s, starting relay\n", conn.RemoteAddr())

		// 获取当前隧道连接
		tunnelMutex.Lock()
		currentTunnelConn := tunnelConn
		tunnelMutex.Unlock()

		// 3. 中继用户 <-> B 隧道
		if currentTunnelConn != nil {
			go relayOneWayA(userConn, currentTunnelConn)
			go relayOneWayA(currentTunnelConn, userConn)
		} else {
			log.Println("[A] no tunnel connection, waiting for B to connect")
		}
	}
}

func relayOneWayA(dst, src net.Conn) {
	defer func() {
		log.Printf("[A] relay direction: %s -> %s completed\n", src.RemoteAddr(), dst.RemoteAddr())
	}()
	io.Copy(dst, src)
}
