.PHONY: build run test tidy lint docker

BINARY=gateway
MAIN=./cmd/gateway

build:
	go build -o bin/$(BINARY) $(MAIN)

run: build
	./bin/$(BINARY) -config config.yaml

test:
	go test ./...

tidy:
	go mod tidy

docker:
	docker build -t ai-gateway:latest .
