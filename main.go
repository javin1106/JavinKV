package main

import (
	"flag"
	"javinkv/config"
	"javinkv/server"
	"log"
)

func setupFlags() {
	flag.StringVar(&config.Host, "host", "0.0.0.0", "host for the JavinKV server")
	flag.IntVar(&config.Port, "port", 7379, "port for the JavinKV server")
	flag.Parse()
}

func main() {
	setupFlags()
	log.Println("JavinKV loading...")
	server.RunSyncTCPServer()
}
