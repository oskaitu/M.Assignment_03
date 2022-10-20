## How to setup proto.grpc.pb.go

	
Remember to use: go get "google.golang.org/grpc"

After setting up .proto file use this:
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/service.proto



