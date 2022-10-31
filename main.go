package main

import (
	"context"
	"log"
	"net"
	"os"
	"sort"
	"sync"
	"time"

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
	stream proto.ChittyChat_JoinServer
	id     string
	name   string
	active bool
	error  chan error
}

type ChatMessage = proto.ChatMessage

type ChatMessageVault struct {
	mu       sync.Mutex
	messages []*ChatMessage
}

type Server struct {
	connections []*Connection
	proto.ChittyChatServer
	message_vault ChatMessageVault
}

func add_message(vault *ChatMessageVault, message *ChatMessage) {
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

func latest_time(vault *ChatMessageVault) int32 {
	vault.mu.Lock()
	defer vault.mu.Unlock()
	if len(vault.messages) == 0 {
		return 0
	}
	return vault.messages[len(vault.messages)-1].Timestamp
}

func send_message(message *ChatMessage, connection *Connection, server *Server) {
	if !connection.active {
		return
	}
	if message.Content == " " || message.Content == "" || len([]rune(message.Content)) > 128 {
		return
	}

	err := connection.stream.Send(message)
	if message.Id != "keep-alive" {
		grpcLog.Info("Sending message to: ", connection.stream, " username: ", connection.name)
	}

	if err != nil {
		drop_connection(connection, server, err)
	}
}

func send_to_all(server *Server, message *ChatMessage) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	for _, connection := range server.connections {
		wait.Add(1)

		go func(message *ChatMessage, connection *Connection) {
			defer wait.Done()
			send_message(message, connection, server)
		}(message, connection)
	}

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done
}

func drop_connection(connection *Connection, server *Server, err error) {
	grpcLog.Errorf("Error with Stream: %s - Error: %v", connection.stream, err)
	grpcLog.Errorf("Dropping connection to: %s", connection.name)
	connection.active = false
	connection.error <- err

	left_message := &ChatMessage{
		Id:              connection.id,
		Content:         "left",
		Timestamp:       latest_time(&server.message_vault),
		Username:        connection.name,
		IsStatusMessage: true,
	}

	// Add message to server "vault"
	add_message(&server.message_vault, left_message)

	send_to_all(server, left_message)
}

func (server *Server) Join(pconn *proto.Connect, stream proto.ChittyChat_JoinServer) error {
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
	server.message_vault.mu.Lock()
	for _, message := range server.message_vault.messages {
		send_message(message, connection, server)
	}
	server.message_vault.mu.Unlock()

	join_message := ChatMessage{
		Id:              user.Id,
		Content:         "joined",
		Timestamp:       latest_time(&server.message_vault),
		Username:        user.Name,
		IsStatusMessage: true,
	}

	add_message(&server.message_vault, &join_message)
	send_to_all(server, &join_message)

	// Keep alive message loop
	go func() {
		for {
			keep_alive_message := ChatMessage{
				Id:              "keep-alive",
				Content:         "keep-alive",
				Timestamp:       latest_time(&server.message_vault),
				Username:        "",
				IsStatusMessage: true,
			}
			send_message(&keep_alive_message, connection, server)
			time.Sleep(time.Second)
		}
	}()

	// Add client connection to pool of connections
	server.connections = append(server.connections, connection)
	return <-connection.error
}

func (server *Server) Publish(context context.Context, message *proto.ChatMessage) (*proto.Close, error) {
	// Add message to server "vault"
	add_message(&server.message_vault, message)

	// Broadcast
	send_to_all(server, message)

	return &proto.Close{}, nil
}

func main() {
	server := &Server{make([]*Connection, 0), proto.UnimplementedChittyChatServer{}, ChatMessageVault{}}

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error creating the server %v", err)
	}

	grpcLog.Info("Starting server at port :8080")

	proto.RegisterChittyChatServer(grpcServer, server)
	grpcServer.Serve(listener)

}
