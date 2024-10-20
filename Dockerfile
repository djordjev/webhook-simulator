FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go build -o ./_out/wh-simulator ./cmd/wh-simulator.go

FROM alpine:latest

LABEL maintainer="Djordje Vukovic"

WORKDIR /app

COPY --from=builder /app/_out/wh-simulator ./wh-simulator



CMD ./wh-simulator

EXPOSE 4488