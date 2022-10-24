FROM golang:alpine as build-env

ENV GO111MODULE=on

RUN apk update && apk add bash ca-certificates git gcc g++ libc-dev

RUN mkdir /docker_folder
RUN mkdir -p /docker_folder/proto

WORKDIR /docker_folder

COPY ./proto/service_grpc.pb.go /docker_folder/proto
COPY ./proto/service.pb.go /docker_folder/proto
COPY ./main.go /docker_folder

COPY go.mod .
COPY go.sum .

RUN go mod download

RUN go build -o docker_folder .

CMD ./docker_folder