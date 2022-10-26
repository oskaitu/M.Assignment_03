package main

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"google.golang.org/grpc"

	proto "github.com/oskaitu/M.Assignment_03/proto"
)

var client proto.BroadcastClient
var wait *sync.WaitGroup
var lamport_time int32

func init() {
	wait = &sync.WaitGroup{}

}

func connect(user *proto.User) error {
	var streamerror error

	stream, err := client.CreateStream(context.Background(), &proto.Connect{
		User:   user,
		Active: true,
	})

	if err != nil {
		return fmt.Errorf("Connection failed :%v", err)
	}

	wait.Add(1)
	go func(str proto.Broadcast_CreateStreamClient) {
		defer wait.Done()

		for {
			msg, err := str.Recv()
			if err != nil {
				streamerror = fmt.Errorf("Error reading message: %v", err)
				break
			}

			fmt.Printf("%s : %s\n", msg.Username, msg.Content)


			if lamport_time < msg.Timestamp {
				lamport_time = msg.Timestamp
			}

			fmt.Printf("%v : %s, at time %d\n", msg.Id, msg.Content, msg.Timestamp)

			lamport_time++

		}

	}(stream)

	return streamerror
}

func main() {
	done := make(chan int)

	name := flag.String("N", "Anon", "The name of the user")
	flag.Parse()

	id := sha256.Sum256([]byte(time.Now().String() + *name))

	conn, err := grpc.Dial(":8080", grpc.WithInsecure())

	if err != nil {
		log.Fatalf("Couldn't connect to service: %v", err)
	}

	client = proto.NewBroadcastClient(conn)
	user := &proto.User{
		Id:   hex.EncodeToString(id[:]),
		Name: *name,
	}

	connect(user)

	wait.Add(1)

	go func() {
		defer wait.Done()

		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			msg := &proto.Message{
				Id:        user.Id,
				Content:   scanner.Text(),
				Timestamp: lamport_time,
        Username: user.Name,
			}

			_, err := client.BroadcastMesssage(context.Background(), msg)
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
