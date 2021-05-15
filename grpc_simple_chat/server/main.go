package main

import (
	"context"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"google.golang.org/grpc"

	"github.com/evg1605/async-lab/chat_core"
	"github.com/evg1605/async-lab/chat_stub_storage"
	"github.com/evg1605/async-lab/grpc_simple_chat/chat_contracts"
)

func main() {
	lis, err := net.Listen("tcp", "localhost:8080")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption

	grpcServer := grpc.NewServer(opts...)
	ctxChat, cancelChat := context.WithCancel(context.Background())
	chatCommands, chatClosed := chat_core.StartCore(ctxChat, chat_stub_storage.NewStorage())

	chat_contracts.RegisterChatServer(grpcServer, newChatServer(ctxChat, chatCommands))

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("try to do graceful stop")
		cancelChat()
		grpcServer.GracefulStop()
		<-chatClosed
	}()

	log.Println("try to start server...")
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed serve: %v", err)
	}

	log.Println("server stopped...")
}
