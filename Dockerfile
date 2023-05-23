FROM golang:latest

WORKDIR /app

COPY main.go /app

RUN go mod init go-server

RUN go mod tidy

RUN go mod download

RUN go build

RUN go run main.go