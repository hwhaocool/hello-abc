package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"sync"
	"time"
)

func startB() {

	checkConfig()

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
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			defer func() {
				log.Printf("[B] relay direction: %s -> %s completed\n", connA.RemoteAddr(), connC.RemoteAddr())
			}()
			relayOneWay(connA, connC)
		}()
		go func() {
			defer wg.Done()
			defer func() {
				log.Printf("[B] relay direction: %s -> %s completed\n", connC.RemoteAddr(), connA.RemoteAddr())
			}()
			relayOneWay(connC, connA)
		}()

		// 等待两个方向都完成
		wg.Wait()
		log.Println("[B] connection lost, reconnecting in 3s...")
		time.Sleep(3 * time.Second)
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
	log.Println("[B] connecting to A")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.A.IP, cfg.A.PortTunnel))
	if err != nil {
		return nil, err
	}
	log.Printf("[B] connected to A from %s\n", conn.RemoteAddr())
	return conn, nil
}

func connectC() (net.Conn, error) {
	log.Println("[B] connecting to C")
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%d", cfg.C.IP, cfg.C.PortTunnel))
	if err != nil {
		return nil, err
	}
	log.Printf("[B] connected to C from %s\n", conn.RemoteAddr())
	return conn, nil
}

func relayOneWay(dst, src net.Conn) {
	io.Copy(dst, src)
}
