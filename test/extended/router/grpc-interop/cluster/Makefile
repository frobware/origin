export GO111MODULE=on

all:
	go build -o client client/client.go
	go build -o server server/server.go
	go mod vendor
	go mod verify
	go mod tidy
