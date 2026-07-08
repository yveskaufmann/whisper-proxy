FROM golang:1.26.4-trixie AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download && go mod verify

COPY . .

RUN CGO_ENABLED=0 GOOS=linux make build 

FROM debian:trixie-slim

WORKDIR /app

RUN apt-get update && apt-get dist-upgrade -y && apt-get install -y --no-install-recommends \
    ffmpeg \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/* 

COPY --from=builder /app/bin/whisper-proxy /app/whisper-proxy

EXPOSE 8080

ENTRYPOINT ["/app/whisper-proxy"]