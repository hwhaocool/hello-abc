package main

import (
	"io"
	"log"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Config struct {
	Role string   `toml:"role"`
	A    SectionA `toml:"a"`
	C    SectionC `toml:"c"`
}

type SectionA struct {
	IP         string `toml:"ip"`
	PortTunnel int    `toml:"port_tunnel"`
	PortServer int    `toml:"port_server"`
}

type SectionC struct {
	IP         string `toml:"ip"`
	PortTunnel int    `toml:"port_tunnel"`
}

func init() {

	// 创建 l
	hook := &lumberjack.Logger{
		Filename:   "app.log", // ⽇志⽂件路径
		MaxSize:    15,        // megabytes MB
		MaxBackups: 3,         // 最多保留2个备份
	}

	// 使用 MultiWriter 同时输出到文件和控制台
	multiWriter := io.MultiWriter(hook, os.Stdout)

	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds | log.Lshortfile)
}
