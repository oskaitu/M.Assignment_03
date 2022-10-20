package main

import (
	"os"

	glog "google.golang.org/grpc/grpclog"
)

var grpcLog glog.LoggerV2

func init(){
	grpcLog = glog.NewLoggerV2(os.Stdout, os.Stdout, os.Stdout)
}

type Connection struct {
	stream proto.Broadcast_CreateStreamServer
	id string
	active bool
	error chan error
}

func main() {

}