package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/evg1605/async-lab/grpc_simple_chat/server"
)

func main() {
	log.Println("try to start server...")
	gracefulStopChat, chatClosed, err := server.StartChatServer("localhost:8080")
	if err != nil {
		log.Fatalf("failed start server: %v", err)
	}
	log.Println("server started...")

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("try to do graceful stop")
		gracefulStopChat()
	}()

	<-chatClosed
	log.Println("server stopped...")
}
