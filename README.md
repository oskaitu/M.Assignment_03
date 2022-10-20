## protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/service.proto

## protoc --go-grpc_out=. --proto_path=proto --proto_path=third_party --go_out=plugins=grpc:proto service.proto