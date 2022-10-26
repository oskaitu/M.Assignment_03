## How to setup proto.grpc.pb.go

	
Remember to use: go get "google.golang.org/grpc"

## After setting up .proto file use this:

protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/service.proto

It will create 2 different pb.go files
Remember to copy both in the docker file. 

COPY ./proto/service_grpc.pb.go /docker_folder/proto
COPY ./proto/service.pb.go /docker_folder/proto
COPY ./main.go /docker_folder

## To build the container:

docker build --tag=test .

## To start the server:

docker run --rm -it  -p 8080:8080 --name chatservice test

## To connect the client (Remember to be inside the client folder):

go run main.go -N "name"



