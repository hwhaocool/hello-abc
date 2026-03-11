package main

import (
	"fmt"
	"log"
	"net"
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
	log.Println("[A] waiting for B connection")

	tunnelConn, err := tunnelLn.Accept()
	if err != nil {
		log.Fatal("[A] Q3 B tunnel connection failed", err)
	}
	log.Println("[A] B connected to tunnel")

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
	log.Println("[A] waiting for user connection")

	userConn, err := userLn.Accept()
	if err != nil {
		log.Fatal("[A] Q6 user connection failed", err)
	}
	log.Println("[A] user connected, starting relay")

	// 3. 中继用户 <-> B 隧道
	Relay(userConn, tunnelConn)
}
