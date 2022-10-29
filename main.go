package main

import (
	"context"
	"log"
	"net"
	"os"
	"sort"
	"sync"

	"google.golang.org/grpc"
	glog "google.golang.org/grpc/grpclog"

	proto "github.com/oskaitu/M.Assignment_03/proto"
)

var grpcLog glog.LoggerV2

func init() {
	grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
}

// TODO: Add name to connection
type Connection struct {
	stream proto.Broadcast_CreateStreamServer
	id     string
	name   string
	active bool
	error  chan error
}

type Message = proto.Message

type MessageVault struct {
	mu       sync.Mutex
	messages []*Message
}

type Server struct {
	connections []*Connection
	proto.BroadcastServer
	message_vault MessageVault
}

func add_message(vault *MessageVault, message *Message) {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	// Append the new message
	vault.messages = append(vault.messages, message)
	// Sort the messages
	sort.Slice(vault.messages, func(i, j int) bool {
		message_one, message_two := vault.messages[i], vault.messages[j]
		if message_one.Timestamp != message_two.Timestamp {
			return message_one.Timestamp < message_two.Timestamp
		}
		return message_one.Id < message_two.Id
	})
}

func latest_time(vault *MessageVault) int32 {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if len(vault.messages) == 0 {
		return 0
	}
	return vault.messages[len(vault.messages)-1].Timestamp
}

func send_message(message *Message, connection *Connection) {
	if !connection.active {
		return
	}
	if message.Content == " " || message.Content == "" || len([]rune(message.Content)) > 128 {
		return
	}

	err := connection.stream.Send(message)
	grpcLog.Info("Sending message to: ", connection.stream, " username:", connection.name)

	if err != nil {
		grpcLog.Errorf("Error with Stream: %s - Error: %v", connection.stream, err)
		connection.active = false
		connection.error <- err
	}
}

func send_to_all(connections []*Connection, message *Message) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, connection := range connections {
		wait.Add(1)

		go func(message *Message, connection *Connection) {
			defer wait.Done()
			send_message(message, connection)
		}(message, connection)
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
}

func (s *Server) CreateStream(pconn *proto.Connect, stream proto.Broadcast_CreateStreamServer) error {
	user := pconn.User

	grpcLog.Info(user.Name, " wants to join the chat")

	connection := &Connection{
		stream: stream,
		id:     user.Id,
		name:   user.Name,
		active: true,
		error:  make(chan error),
	}

	// Get client up to date & send "user join" message
	s.message_vault.mu.Lock()
	for _, message := range s.message_vault.messages {
		send_message(message, connection)
	}
	s.message_vault.mu.Unlock()

	join_message := Message{
		Id:              user.Id,
		Content:         "joined",
		Timestamp:       latest_time(&s.message_vault),
		Username:        user.Name,
		IsStatusMessage: true,
	}

	add_message(&s.message_vault, &join_message)
	send_to_all(s.connections, &join_message)

	// Add client connection to pool of connections
	s.connections = append(s.connections, connection)
	return <-connection.error
}

func (s *Server) BroadcastMesssage(context context.Context, message *proto.Message) (*proto.Close, error) {
	// Add message to server "vault"
	add_message(&s.message_vault, message)

	// Broadcast
	send_to_all(s.connections, message)

	return &proto.Close{}, nil
}

func main() {
	server := &Server{make([]*Connection, 0), proto.UnimplementedBroadcastServer{}, MessageVault{}}

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error creating the server %v", err)
	}

	grpcLog.Info("Starting server at port :8080")

	proto.RegisterBroadcastServer(grpcServer, server)
	grpcServer.Serve(listener)

}
