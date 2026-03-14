package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// RoleC 管理 C 角色的所有状态
type RoleC struct {
	tunnelConn   net.Conn
	forwardConn  net.Conn
	tunnelMu     sync.Mutex
	forwardMu    sync.Mutex
	forwardReady sync.Cond

	// 用于通知连接断开事件
	forwardDisconnected chan struct{}
}

func startC() {
	checkConfigC()

	rc := &RoleC{
		forwardDisconnected: make(chan struct{}, 1),
	}
	rc.forwardReady.L = &rc.forwardMu

	// 启动转发连接管理器
	go rc.manageForward()

	// 监听来自 B 的隧道连接
	tunnelLn, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.C.PortTunnel))
	if err != nil {
		log.Fatal("[C] failed to listen tunnel port", err)
	}
	defer tunnelLn.Close()
	log.Printf("[C] waiting for B connection at port %d\n", cfg.C.PortTunnel)

	// 主循环：持续接受隧道连接
	for {
		conn, err := tunnelLn.Accept()
		if err != nil {
			log.Println("[C] accept connection failed", err)
			continue
		}

		go rc.handleTunnel(conn)
	}
}

func checkConfigC() {
	if cfg.C.PortTunnel == 0 {
		log.Fatalln("[C] tunnel port not set")
	}

	if cfg.C.PortForward == 0 {
		log.Fatalln("[C] forward port not set")
	}
}

// manageForward 管理转发目标连接
func (rc *RoleC) manageForward() {
	for {
		// 连接转发目标
		forwardConn, err := connectForward()
		if err != nil {
			log.Println("[C] failed to connect forward, retrying in 1s", err)
			sleep(1000)
			continue
		}

		rc.forwardMu.Lock()
		oldConn := rc.forwardConn
		rc.forwardConn = forwardConn
		addr := forwardConn.RemoteAddr().String()
		rc.forwardReady.Broadcast()
		rc.forwardMu.Unlock()

		if oldConn != nil {
			log.Println("[C] closing old forward connection")
			oldConn.Close()
		}
		log.Printf("[C] forward connected from %s\n", addr)

		// 等待断开信号
		<-rc.forwardDisconnected
		log.Println("[C] forward disconnection requested")

		rc.forwardMu.Lock()
		if rc.forwardConn == forwardConn {
			rc.forwardConn = nil
		}
		forwardConn.Close()
		rc.forwardMu.Unlock()
	}
}

// handleTunnel 处理单个隧道连接
func (rc *RoleC) handleTunnel(tunnelConn net.Conn) {
	log.Printf("[C] new tunnel connection from %s\n", tunnelConn.RemoteAddr())
	defer tunnelConn.Close()

	rc.tunnelMu.Lock()
	oldConn := rc.tunnelConn
	rc.tunnelConn = tunnelConn
	rc.tunnelMu.Unlock()

	if oldConn != nil {
		log.Println("[C] closing old tunnel connection")
		oldConn.Close()
	}

	// 等待转发连接就绪
	rc.forwardMu.Lock()
	for rc.forwardConn == nil {
		log.Println("[C] waiting for forward connection...")
		rc.forwardReady.Wait()
	}
	currentForwardConn := rc.forwardConn
	rc.forwardMu.Unlock()

	log.Println("[C] starting relay tunnel <-> forward")

	// 中继数据
	done := make(chan connDirection, 2)

	go func() {
		relayOneWayC(tunnelConn, currentForwardConn)
		done <- dirTunnel
	}()

	go func() {
		relayOneWayC(currentForwardConn, tunnelConn)
		done <- dirForward
	}()

	// 等待第一个完成的方向
	direction := <-done
	log.Printf("[C] relay stopped, direction: %d\n", direction)

	// 立即关闭两个连接
	tunnelConn.Close()
	currentForwardConn.Close()

	// 通知对应的连接管理器
	switch direction {
	case dirForward:
		select {
		case rc.forwardDisconnected <- struct{}{}:
		default:
		}
	case dirTunnel:
		// 隧道断开，不需要特别处理
	}
}

func connectForward() (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.C.PortForward))
}

func relayOneWayC(dst, src net.Conn) {
	defer func() {
		log.Printf("[C] relay direction: %s -> %s completed\n", src.RemoteAddr(), dst.RemoteAddr())
	}()
	io.Copy(dst, src)
}
