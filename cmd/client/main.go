package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"
	"golang.org/x/exp/rand"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "chat-app/proto/pb"
)

var (
	client   pb.ChatServiceClient
	username string
	chatRoom string
)

func main() {
	conn, err := grpc.Dial("localhost:50051", grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("Failed to connect to gRPC server: %v", err)
	}
	defer conn.Close()

	client = pb.NewChatServiceClient(conn)

	a := app.New()
	w := a.NewWindow("Chat Application")

	userEntry := widget.NewEntry()
	userEntry.SetPlaceHolder("Enter Username")

	chatRoomEntry := widget.NewEntry()
	chatRoomEntry.SetPlaceHolder("Enter Chat Room ID")

	messageEntry := widget.NewEntry()
	messageEntry.SetPlaceHolder("Enter Message")

	messageList := widget.NewList(
		func() int {
			return len(messages)
		},
		func() fyne.CanvasObject {
			return widget.NewLabel("template")
		},
		func(i widget.ListItemID, o fyne.CanvasObject) {
			parts := strings.SplitN(messages[i], ": ", 2)
			label := o.(*widget.Label)

			if parts[0] == username{
				label.Alignment = fyne.TextAlignTrailing
				label.SetText(messages[i])
			} else {
				label.Alignment = fyne.TextAlignLeading
				label.SetText(messages[i])
			}
		},
	)

	sendButton := widget.NewButton("Send", func() {
		message := messageEntry.Text
		sendMessage(message)
		messageEntry.SetText("")
	})

	joinButton := widget.NewButton("Join Chat Room", func() {
		username = userEntry.Text
		chatRoom = chatRoomEntry.Text
			// Generate a unique chat room ID if not provided by the client
		if chatRoom == "" {
			chatRoom = generateID()
			chatRoomEntry.SetText(chatRoom)
		}

		joinChatRoom(username, chatRoom)
		go streamMessages(messageList)
		
	})

	scrollContainer := container.NewVScroll(messageList)
	scrollContainer.SetMinSize(fyne.NewSize(600, 300))

	w.SetContent(container.NewVBox(
		userEntry,
		chatRoomEntry,
		messageEntry,
		sendButton,
		joinButton,
		scrollContainer,
	))

	w.Resize(fyne.NewSize(400, 600))
	w.ShowAndRun()
}

// Function to join a chat room
func joinChatRoom(username, chatRoomID string) {
	user := &pb.User{
		Id:   username,
		Name: username,
	}
	req := &pb.JoinRequest{
		ChatRoomId: chatRoomID,
		User:       user,
	}
	_, err := client.Join(context.Background(), req)
	if err != nil {
		log.Printf("Failed to join chat room: %v", err)
	}
}

var messages []string

func sendMessage(content string) {
	msg := &pb.Message{
		UserId:    username,
		Content:   content,
		Timestamp: time.Now().Unix(),
		ChatRoomId: chatRoom,
	}
	_, err := client.SendMessage(context.Background(), msg)
	if err != nil {
		log.Printf("Failed to send message: %v", err)
	}
}

func streamMessages(messageList *widget.List) {
	stream, err := client.StreamMessages(context.Background(), &pb.ChatRoom{Id: chatRoom})
	if err != nil {
		log.Fatalf("Error establishing message stream: %v", err)
	}
	for {
		in, err := stream.Recv()
		if err != nil {
			log.Fatalf("Failed to receive message: %v", err)
		}
		messages = append(messages, fmt.Sprintf("%s: %s", in.UserId, in.Content))
		messageList.Refresh()
	}
}


// generateID generates a random ID for users and messages.
func generateID() string {
	rand.Seed(uint64(time.Now().UnixNano()))
	return fmt.Sprintf("%d", rand.Intn(1000000))
}