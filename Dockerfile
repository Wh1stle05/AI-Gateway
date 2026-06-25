FROM golang:1.22-alpine AS builder

WORKDIR /src
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /gateway ./cmd/gateway

FROM alpine:3.20

RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /gateway /app/gateway
COPY config.example.yaml /app/config.example.yaml

EXPOSE 8080
ENTRYPOINT ["/app/gateway"]
CMD ["-config", "/app/config.yaml"]
