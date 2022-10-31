package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"

	proto "github.com/oskaitu/M.Assignment_03/proto"
)

var client proto.ChittyChatClient
var wait *sync.WaitGroup
var lamport_clock LamportClock
var name = "Anon"

type ChatMessage = proto.ChatMessage

type LamportClock struct {
	mu   sync.Mutex
	time int32
}

func set_time(clock *LamportClock, value int32) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.time = value
}

func update_time(clock *LamportClock) {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	clock.time++
}

func get_time(clock *LamportClock) int32 {
	clock.mu.Lock()
	defer clock.mu.Unlock()
	return clock.time
}

func init() {
	wait = &sync.WaitGroup{}
}

func connect(user *proto.User) error {
	var streamerror error

	stream, err := client.Join(context.Background(), &proto.Connect{
		User: user,
	})

	if err != nil {
		return fmt.Errorf("Connection failed :%v", err)
	}

	wait.Add(1)
	go func(stream proto.ChittyChat_JoinClient) {
		defer wait.Done()

		for {
			message, err := stream.Recv()
			if err != nil {
				streamerror = fmt.Errorf("Error reading message: %v", err)
				break
			}

			if message.Id == user.Id {
				continue
			}

			if message.Id == "keep-alive" {
				continue
			}

			if lamport_clock.time < message.Timestamp {
				set_time(&lamport_clock, message.Timestamp)
			}

			// Write message to console
			fmt.Printf("t %d --- ", message.Timestamp)

			if message.IsStatusMessage {
				switch {
				case message.Content == "joined":
					fmt.Printf("%s has joined the chat.", message.Username)
				case message.Content == "left":
					fmt.Printf("%s has left the chat.", message.Username)
				}
			} else {
				fmt.Printf("%s : \"%s\"", message.Username, message.Content)
			}

			fmt.Printf("\n")

			if !message.IsStatusMessage {
				update_time(&lamport_clock)
			}
		}

	}(stream)

	return streamerror
}

func get_outbound_IP() net.IP {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)

	return localAddr.IP
}

func main() {
	done := make(chan int)

	/* fmt.Print("Enter username: ")
	scannerName := bufio.NewScanner(os.Stdin)
	for scannerName.Scan() {
		text := scannerName.Text()

		name = text
	} */

	fmt.Print("Enter username: ")
	var userinput string
	fmt.Scanf("%s", &userinput)
	name = userinput

	local_ip := get_outbound_IP()
	id := sha256.Sum256([]byte(time.Now().String() + name))
	conn, err := grpc.Dial(local_ip.String()+":8080", grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Couldn't connect to service: %v", err)
	}

	client = proto.NewChittyChatClient(conn)
	user := &proto.User{
		Id:   hex.EncodeToString(id[:]),
		Name: name,
	}

	connect(user)

	wait.Add(1)

	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			text := scanner.Text()

			update_time(&lamport_clock)

			msg := &ChatMessage{

				Id:              user.Id,
				Content:         text,
				Timestamp:       get_time(&lamport_clock),
				Username:        user.Name,
				IsStatusMessage: false,
			}

			_, err := client.Publish(context.Background(), msg)
			if err != nil {
				fmt.Printf("Error sending Message: %v", err)
				break
			}
		}
	}()

	go func() {
		wait.Wait()
		close(done)
	}()

	<-done

}
