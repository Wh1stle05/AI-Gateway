.PHONY: build run test tidy lint fmt vet docker loadtest clean

BINARY=gateway
MAIN=./cmd/gateway

build:
	go build -o bin/$(BINARY) $(MAIN)

run: build
	./bin/$(BINARY) -config config.yaml

run-loadtest: build
	./bin/$(BINARY) -config config.loadtest.yaml

test:
	go test -race ./...

coverage:
	go test -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

fmt:
	gofmt -s -w .

vet:
	go vet ./...

lint:
	golangci-lint run ./...

tidy:
	go mod tidy

docker:
	docker build -t ai-gateway:latest .

loadtest:
	k6 run scripts/loadtest.js

clean:
	rm -rf bin/ coverage.out coverage.html
