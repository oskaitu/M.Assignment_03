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

og
docker build --tag=test .


docker build -f dockerfile.server --tag=server .
docker build -f dockerfile.client --tag=client .


## To start the server:

docker run --rm -it  -p 8080:8080 --name chatservice server

remember to run server on a public network (not set to private)

## To connect the client (Remember to be inside the client folder):

go run main.go
OR
docker run --rm  -it client
fyi the clint.go grpcdial needs to be changed to whatever your ipv4 address is. so to make the client run as docker container you have to hardcode that as line 142 in client.go 

set client.go contain host IP address 
check ipconfig /all under IPv4 
host can just have :8080 in their client.go



