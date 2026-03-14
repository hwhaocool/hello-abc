package main

import (
	"fmt"
	"log"
	"net"
	"sync"
)

// RoleA 管理 A 角色的所有状态
type RoleA struct {
	tunnelConn  net.Conn
	userConn    net.Conn
	tunnelMu    sync.Mutex
	userMu      sync.Mutex
	tunnelReady sync.Cond
	userReady   sync.Cond

	// 用于通知连接断开事件
	tunnelDisconnected chan struct{}
	userDisconnected   chan struct{}
}

func startA() {
	checkConfigA()

	ra := &RoleA{
		tunnelDisconnected: make(chan struct{}, 1),
		userDisconnected:   make(chan struct{}, 1),
	}
	ra.tunnelReady.L = &ra.tunnelMu
	ra.userReady.L = &ra.userMu

	// 启动隧道连接管理器
	go ra.manageTunnel()

	// 监听用户连接
	userLn, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.A.PortServer))
	if err != nil {
		log.Fatal("[A] failed to listen server port", err)
	}
	defer userLn.Close()
	log.Printf("[A] waiting for user connection at port %d\n", cfg.A.PortServer)

	// 主循环：持续接受用户连接
	for {
		conn, err := userLn.Accept()
		if err != nil {
			log.Println("[A] accept user connection failed", err)
			continue
		}

		go ra.handleUser(conn)
	}
}

func checkConfigA() {
	if cfg.A.PortTunnel == 0 {
		log.Fatalln("[A] tunnel port not set")
	}
	if cfg.A.PortServer == 0 {
		log.Fatalln("[A] server port not set")
	}
}

// manageTunnel 管理来自 B 的隧道连接
func (ra *RoleA) manageTunnel() {
	// 监听来自 B 的隧道连接
	tunnelLn, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.A.PortTunnel))
	if err != nil {
		log.Fatal("[A] failed to listen tunnel port", err)
	}
	defer tunnelLn.Close()
	log.Printf("[A] waiting for B connection at port %d\n", cfg.A.PortTunnel)

	for {
		conn, err := tunnelLn.Accept()
		if err != nil {
			log.Println("[A] accept B connection failed", err)
			continue
		}

		go ra.handleTunnel(conn)
	}
}

// handleTunnel 处理单个隧道连接
func (ra *RoleA) handleTunnel(conn net.Conn) {
	log.Printf("[A] new B connection from %s\n", conn.RemoteAddr())

	ra.tunnelMu.Lock()
	oldConn := ra.tunnelConn
	ra.tunnelConn = conn
	ra.tunnelReady.Broadcast()
	ra.tunnelMu.Unlock()

	if oldConn != nil {
		log.Println("[A] closing old tunnel connection")
		oldConn.Close()
	}

	// 等待断开信号
	<-ra.tunnelDisconnected
	log.Println("[A] tunnel disconnection requested")

	ra.tunnelMu.Lock()
	if ra.tunnelConn == conn {
		ra.tunnelConn = nil
	}
	conn.Close()
	ra.tunnelMu.Unlock()
}

// handleUser 处理用户连接
func (ra *RoleA) handleUser(conn net.Conn) {
	log.Printf("[A] new user connection from %s\n", conn.RemoteAddr())

	ra.userMu.Lock()
	oldConn := ra.userConn
	ra.userConn = conn
	ra.userReady.Broadcast()
	ra.userMu.Unlock()

	if oldConn != nil {
		log.Println("[A] closing old user connection")
		oldConn.Close()
	}

	// 等待隧道连接就绪
	ra.tunnelMu.Lock()
	for ra.tunnelConn == nil {
		log.Println("[A] waiting for tunnel connection...")
		ra.tunnelReady.Wait()
	}
	currentTunnelConn := ra.tunnelConn
	ra.tunnelMu.Unlock()

	log.Println("[A] starting relay user <-> tunnel")

	// 中继数据
	done := make(chan connDirection, 2)
	stopRelay := make(chan struct{}, 1)

	go func() {
		relayOneWayA(conn, currentTunnelConn, stopRelay)
		done <- dirUser
	}()

	go func() {
		relayOneWayA(currentTunnelConn, conn, stopRelay)
		done <- dirTunnel
	}()

	// 等待第一个完成的方向
	direction := <-done
	log.Printf("[A] relay stopped, direction: %d\n", direction)

	// 发送停止信号，确保另一个方向的 relay 也退出
	stopRelay <- struct{}{}

	switch direction {
	case dirTunnel:
		// 隧道断开，关闭用户连接并通知隧道管理器
		conn.Close()
		select {
		case ra.tunnelDisconnected <- struct{}{}:
		default:
		}
	case dirUser:
		// 用户断开，只关闭用户连接，保持隧道连接存活
		conn.Close()
		log.Println("[A] user disconnected, tunnel connection kept alive")
	}
}

func relayOneWayA(dst, src net.Conn, stopRelay chan struct{}) {
	defer func() {
		log.Printf("[A] relay direction: %s -> %s completed\n", src.RemoteAddr(), dst.RemoteAddr())
	}()

	// 使用 buffered reader 和 writer 实现 io.Copy，支持中断
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-stopRelay:
			log.Println("[A] relay stopped by signal")
			return
		default:
			n, err := src.Read(buf)
			if err != nil {
				return
			}
			if n > 0 {
				_, writeErr := dst.Write(buf[:n])
				if writeErr != nil {
					return
				}
			}
		}
	}
}
