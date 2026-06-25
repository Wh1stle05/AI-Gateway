.PHONY: build run test tidy lint docker loadtest

BINARY=gateway
MAIN=./cmd/gateway

build:
	go build -o bin/$(BINARY) $(MAIN)

run: build
	./bin/$(BINARY) -config config.yaml

run-loadtest: build
	./bin/$(BINARY) -config config.loadtest.yaml

test:
	go test ./...

tidy:
	go mod tidy

docker:
	docker build -t ai-gateway:latest .

loadtest:
	k6 run scripts/loadtest.js
