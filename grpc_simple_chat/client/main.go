package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"google.golang.org/grpc"
	"log"
	"os"
	"strings"
	"time"

	"github.com/evg1605/async-lab/grpc_simple_chat/chat_contracts"
)

func main() {
	//flag.String()

	flag.Parse()

	var opts []grpc.DialOption

	opts = append(opts, grpc.WithInsecure())
	conn, err := grpc.Dial("localhost:8080", opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer func(conn *grpc.ClientConn) {
		_ = conn.Close()
	}(conn)

	chat := chat_contracts.NewChatClient(conn)
	runManualSendMessages(chat, "user-1")
	//
	//go sendMessageLoop(chat)
	//time.Sleep(time.Second * 600)

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

func runManualSendMessages(chat chat_contracts.ChatClient, userID string) {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("Enter message (empty for exit): ")
		content, err := reader.ReadString('\n')
		if err != nil {
			return
		}
		content = strings.TrimRight(content, "\n")
		if content == "" {
			return
		}
		msg, err := chat.SendMessage(context.Background(), &chat_contracts.SendMessageCmd{
			UserID:  userID,
			Content: content,
		})
		if err != nil {
			log.Printf("error send message: %s", err)
		} else {
			log.Printf("send message ok: %v, %s, %s", msg.Id, msg.UserID, msg.Content)
		}
	}
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
