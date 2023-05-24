FROM golang:latest

WORKDIR /app

ENV GOPRIVATE "github.com"
COPY go.mod .
COPY go.sum .
RUN go mod download

COPY main.go .
COPY ./static/ ./static/
RUN go build -o TodoApp

CMD [ "./TodoApp" ]