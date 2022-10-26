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

type Server struct {
	Connection []*Connection
	proto.BroadcastServer
	message_vault MessageVault
}

type Message = proto.Message

type MessageVault struct {
	mu       sync.Mutex
	messages []*Message
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

func (s *Server) CreateStream(pconn *proto.Connect, stream proto.Broadcast_CreateStreamServer) error {
	connection := &Connection{
		stream: stream,
		id:     pconn.User.Id,
		name:   pconn.User.Name,
		active: true,
		error:  make(chan error),
	}

	// Get client up to date
	s.message_vault.mu.Lock()
	for _, message := range s.message_vault.messages {
		send_message(message, connection)
	}
	s.message_vault.mu.Unlock()

	// Add client connection to pool of connections
	s.Connection = append(s.Connection, connection)

	return <-connection.error
}

func (s *Server) BroadcastMesssage(context context.Context, message *proto.Message) (*proto.Close, error) {
	wait := sync.WaitGroup{}
	done := make(chan int)

	// Add message to server "vault"
	add_message(&s.message_vault, message)

	for _, connection := range s.Connection {
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
	return &proto.Close{}, nil
}

func main() {
	var connections []*Connection
	server := &Server{connections, proto.UnimplementedBroadcastServer{}, MessageVault{}}

	grpcServer := grpc.NewServer()
	listener, err := net.Listen("tcp", ":8080")
	if err != nil {
		log.Fatalf("Error creating the server %v", err)
	}

	grpcLog.Info("Starting server at port :8080")

	proto.RegisterBroadcastServer(grpcServer, server)
	grpcServer.Serve(listener)

}
