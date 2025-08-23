package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/codecrafters-io/redis-starter-go/app/server"
)

func main() {
	port := flag.Int("port", 6379, "Port to listen on")
	flag.Parse()

	address := fmt.Sprintf("0.0.0.0:%d", *port)

	newServer, err := server.NewServer(address)
	if err != nil {
		fmt.Printf("Failed to create server: %v\n", err)
		os.Exit(1)
	}

	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		newServer.Start()
	}()

	fmt.Println("Redis server is running. Press Ctrl+C to stop.")

	<-signalChan

	newServer.Stop()
}
