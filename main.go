package main

import (
	"log"

	"github.com/BurntSushi/toml"
	"github.com/fatih/color"
)

var cfg Config

var Version = "dev"

func init() {
	_, err := toml.DecodeFile("config.toml", &cfg)
	if err != nil {
		log.Fatalf("Failed to read config.toml: %v", err)
	}
}

func main() {
	color.Magenta("version %s, Role: %s\n", Version, cfg.Role)

	switch cfg.Role {
	case "a":
		startA()
	case "b":
		startB()
	case "c":
		startC()
	default:
		log.Fatalf("invalid role")
	}

}
