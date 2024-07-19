package main

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	pb "chat-app/proto/pb" // Import the generated protobuf code
)

// server is used to implement pb.ChatServiceServer.
type server struct {
	pb.UnimplementedChatServiceServer
	mu        sync.Mutex                                       // Mutex to protect concurrent access
	chatRooms map[string][]*pb.User                            // Map to store chat rooms and their users
	streams   map[string][]pb.ChatService_StreamMessagesServer // Map to store streams for each chat room
}

// NewServer creates a new server instance.
func NewServer() *server {
	return &server{
		chatRooms: make(map[string][]*pb.User),
		streams:   make(map[string][]pb.ChatService_StreamMessagesServer),
	}
}

// generateID generates a random ID for users and messages.
func generateID() string {
	rand.Seed(time.Now().UnixNano())
	return fmt.Sprintf("%d", rand.Intn(1000000))
}

// Join allows a user to join a chat room.
func (s *server) Join(ctx context.Context, req *pb.JoinRequest) (*pb.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chatRoomID := req.ChatRoomId
	user := req.User

	// Check if the user is already in the chat room
	for _, u := range s.chatRooms[chatRoomID] {
		if u.Id == user.Id {
			log.Printf("User %s already joined room %s", user.Name, chatRoomID)
			return nil, fmt.Errorf("user already joined the room")
		}
	}

	// Add the user to the chat room
	s.chatRooms[chatRoomID] = append(s.chatRooms[chatRoomID], user)
	log.Printf("User %s joined chat room %s", user.Name, chatRoomID)
	return user, nil
}

// SendMessage sends a message to all users in the specified chat room.
func (s *server) SendMessage(ctx context.Context, msg *pb.Message) (*pb.Message, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	chatRoomID := msg.ChatRoomId
	// Send the message to all streams in the specified chat room
	if streams, ok := s.streams[chatRoomID]; ok {
		for _, stream := range streams {
			if err := stream.Send(msg); err != nil {
				log.Printf("Error sending message to stream: %v", err)
			}
		}
	}

	log.Printf("Message from user %s in chat room %s: %s", msg.UserId, chatRoomID, msg.Content)
	return msg, nil
}

// StreamMessages streams messages from the specified chat room to the client.

func (s *server) StreamMessages(chatRoom *pb.ChatRoom, stream pb.ChatService_StreamMessagesServer) error {
	s.mu.Lock()

	// Check if the stream is already in the list
	for _, existingStream := range s.streams[chatRoom.Id] {
		if existingStream == stream {
			s.mu.Unlock()
			return nil
		}
	}

	// Add stream to the chat room's streams slice
	s.streams[chatRoom.Id] = append(s.streams[chatRoom.Id], stream)
	s.mu.Unlock()

	<-stream.Context().Done()

	s.mu.Lock()
	// Remove the stream from the slice
	for i, existingStream := range s.streams[chatRoom.Id] {
		if existingStream == stream {
			s.streams[chatRoom.Id] = append(s.streams[chatRoom.Id][:i], s.streams[chatRoom.Id][i+1:]...)
			break
		}
	}
	s.mu.Unlock()

	log.Printf("User left chat room %s", chatRoom.Id)
	return nil
}


// func (s *server) StreamMessages(chatRoom *pb.ChatRoom, stream pb.ChatService_StreamMessagesServer) error {
// 	s.mu.Lock()
// 	// Add the stream to the list of streams for the specified chat room
// 	s.streams[chatRoom.Id] = append(s.streams[chatRoom.Id], stream)
// 	s.mu.Unlock()

// 	<-stream.Context().Done() // Wait until the stream is done

// 	s.mu.Lock()
// 	// Remove the stream from the list of streams for the specified chat room
// 	for _, s := range s.streams[chatRoom.Id] {
// 		if s == stream {
// 			// Commented line shows how to remove the stream from the slice
// 			// s.streams[chatRoom.Id] = append(s.streams[chatRoom.Id][:i], s.streams[chatRoom.Id][i+1:]...)
// 			break
// 		}
// 	}
// 	s.mu.Unlock()

// 	log.Printf("User left chat room %s", chatRoom.Id)
// 	return nil
// }

func main() {
	// Create a listener on TCP port 50051
	listener, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	// Create a new gRPC server
	s := grpc.NewServer()
	// Register the chat service with the server
	pb.RegisterChatServiceServer(s, NewServer())
	// Register reflection service on gRPC server (for debugging purposes)
	reflection.Register(s)

	log.Println("gRPC server is running on port 50051")
	// Serve gRPC server
	if err := s.Serve(listener); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}
