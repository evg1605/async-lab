package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"

	"github.com/evg1605/async-lab/grpc_simple_chat/chat_contracts"
)

func main() {

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())
	//opts = append(opts, grpc.WithBlock())
	conn, err := grpc.Dial("localhost:8080", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	chat := chat_contracts.NewChatClient(conn)

	go sendMessageLoop(chat)
	time.Sleep(time.Second * 600)

	//messages, err := chat.GetMessages(context.Background(), &chat_contracts.GetMessagesCmd{
	//	LastMsgID:         -1,
	//	LastMessagesCount: 0,
	//})
	//if err != nil {
	//	log.Fatalf("error get messages: %v", err)
	//}
	//for {
	//	mm, err := messages.Recv()
	//	if err != nil {
	//		log.Fatalf("error recive  next messages: %v", err)
	//	}
	//	for _, m := range mm.Messages {
	//		log.Println(m.Content)
	//	}
	//}
}

func sendMessageLoop(chat chat_contracts.ChatClient) {
	userName := "user1"
	for i := int64(0); ; i++ {
		msg, err := chat.SendMessage(context.Background(), &chat_contracts.SendMessageCmd{
			UserID:  userName,
			Content: fmt.Sprintf("message %v", i),
		})
		if err != nil {
			log.Printf("error send message: %v", err)
		} else {
			log.Printf("success send message: %v", msg.Id)
		}
		time.Sleep(time.Second * 2)
	}
}
