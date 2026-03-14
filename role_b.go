package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

// RoleB 管理 B 角色的所有状态
type RoleB struct {
	connA     net.Conn
	connC     net.Conn
	connAMu   sync.Mutex
	connCMu   sync.Mutex
	connAReady sync.Cond
	connCReady sync.Cond

	// 用于通知连接断开事件
	aDisconnected chan struct{}
	cDisconnected chan struct{}
}

func startB() {
	checkConfig()

	rb := &RoleB{
		aDisconnected: make(chan struct{}, 1),
		cDisconnected: make(chan struct{}, 1),
	}
	rb.connAReady.L = &rb.connAMu
	rb.connCReady.L = &rb.connCMu

	// 启动 A 连接管理器
	go rb.manageA()

	// 启动 C 连接管理器
	go rb.manageC()

	// 主中继循环
	rb.runRelay()
}

// manageA 管理 A 的连接
func (rb *RoleB) manageA() {
	for {
		// 1. 连接 A
		connA, err := connectA()
		if err != nil {
			log.Println("[B] failed to connect A, retrying in 1s", err)
			sleep(1000)
			continue
		}

		// 2. 更新连接
		rb.connAMu.Lock()
		oldConn := rb.connA
		rb.connA = connA
		addr := connA.RemoteAddr().String()
		rb.connAMu.Unlock()

		if oldConn != nil {
			log.Println("[B] closing old A connection")
			oldConn.Close()
		}
		log.Printf("[B] A connected from %s\n", addr)

		// 3. 通知等待者
		rb.connAReady.Broadcast()

		// 4. 等待 runRelay 发来的断开信号
		<-rb.aDisconnected
		log.Println("[B] A disconnection requested")

		// 5. 清理连接
		rb.connAMu.Lock()
		if rb.connA == connA {
			rb.connA = nil
		}
		connA.Close()
		rb.connAMu.Unlock()
	}
}

// manageC 管理 C 的连接
func (rb *RoleB) manageC() {
	for {
		// 1. 连接 C
		connC, err := connectC()
		if err != nil {
			log.Println("[B] failed to connect C, retrying in 1s", err)
			sleep(1000)
			continue
		}

		// 2. 更新连接
		rb.connCMu.Lock()
		oldConn := rb.connC
		rb.connC = connC
		addr := connC.RemoteAddr().String()
		rb.connCMu.Unlock()

		if oldConn != nil {
			log.Println("[B] closing old C connection")
			oldConn.Close()
		}
		log.Printf("[B] C connected from %s\n", addr)

		// 3. 通知等待者
		rb.connCReady.Broadcast()

		// 4. 等待 runRelay 发来的断开信号
		<-rb.cDisconnected
		log.Println("[B] C disconnection requested")

		// 5. 清理连接
		rb.connCMu.Lock()
		if rb.connC == connC {
			rb.connC = nil
		}
		connC.Close()
		rb.connCMu.Unlock()
	}
}

// runRelay 运行中继循环
func (rb *RoleB) runRelay() {
	for {
		// 等待两个连接都就绪
		rb.connAMu.Lock()
		for rb.connA == nil {
			rb.connAReady.Wait()
		}
		currentConnA := rb.connA
		rb.connAMu.Unlock()

		rb.connCMu.Lock()
		for rb.connC == nil {
			rb.connCReady.Wait()
		}
		currentConnC := rb.connC
		rb.connCMu.Unlock()

		log.Println("[B] starting relay A <-> C")

		// 任意一个方向完成就通知对应的连接管理器
		done := make(chan connDirection, 2)

		go func() {
			relayOneWay(currentConnA, currentConnC)
			done <- dirA
		}()

		go func() {
			relayOneWay(currentConnC, currentConnA)
			done <- dirC
		}()

		// 等待第一个完成的方向
		direction := <-done
		log.Printf("[B] relay stopped, direction: %d\n", direction)

		// 立即关闭两个连接，加速中继退出
		currentConnA.Close()
		currentConnC.Close()

		// 通知对应的连接管理器重连
		switch direction {
		case dirA:
			select {
			case rb.aDisconnected <- struct{}{}:
			default:
			}
		case dirC:
			select {
			case rb.cDisconnected <- struct{}{}:
			default:
			}
		}
	}
}

func checkConfig() {
	if cfg.A.IP == "" {
		log.Fatalln("[B] Q3, ip of A not set")
	}

	if cfg.A.PortTunnel == 0 {
		log.Fatalln("[B] Q4, tunnel port of A not set")
	}

	if cfg.C.IP == "" {
		log.Fatalln("[B] Q5, ip of C not set")
	}

	if cfg.C.PortTunnel == 0 {
		log.Fatalln("[B] Q6, tunnel port of C not set")
	}
}

func connectA() (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.A.IP, cfg.A.PortTunnel))
}

func connectC() (net.Conn, error) {
	return net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.C.IP, cfg.C.PortTunnel))
}

func relayOneWay(dst, src net.Conn) {
	io.Copy(dst, src)
}
