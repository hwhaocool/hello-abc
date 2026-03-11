package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"time"
)

func startB() {
	for {
		// 连接 A
		connA, err := connectA()
		if err != nil {
			log.Println("[B] Q1 failed to connect A, retrying in 3s", err)
			time.Sleep(3 * time.Second)
			continue
		}
		defer connA.Close()

		// 连接 C
		connC, err := connectC()
		if err != nil {
			log.Println("[B] Q2 failed to connect C, retrying in 3s", err)
			time.Sleep(3 * time.Second)
			continue
		}
		defer connC.Close()

		log.Println("[B] starting relay A <-> C")

		// 中继数据
		done := make(chan struct{})
		go func() {
			defer close(done)
			if _, err := relayOneWay(connA, connC); err != nil {
				log.Printf("[B] relay A->C error: %v", err)
			}
		}()
		go func() {
			defer close(done)
			if _, err := relayOneWay(connC, connA); err != nil {
				log.Printf("[B] relay C->A error: %v", err)
			}
		}()

		// 等待任一方向断开
		<-done
		log.Println("[B] connection lost, reconnecting in 3s...")
		time.Sleep(3 * time.Second)
	}
}

func connectA() (net.Conn, error) {
	log.Println("[B] connecting to A")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.A.IP, cfg.A.PortTunnel))
	if err != nil {
		return nil, err
	}
	log.Println("[B] connected to A")
	return conn, nil
}

func connectC() (net.Conn, error) {
	log.Println("[B] connecting to C")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.C.IP, cfg.C.PortTunnel))
	if err != nil {
		return nil, err
	}
	log.Println("[B] connected to C")
	return conn, nil
}

func relayOneWay(dst, src net.Conn) (int64, error) {
	defer dst.Close()
	defer src.Close()
	return io.Copy(dst, src)
}
