.PHONY: all build clean distclean run-server run-client

all: build

go.mod:
	go mod init server
	go mod tidy

build: go.mod
	go mod tidy
	go build -o server server.go
	go build -o client client.go

clean:
	rm -f server client

distclean: clean
	rm -f go.mod go.sum

run-server:
	go run server.go

run-client:
	go run client.go 127.0.0.1 joguinho