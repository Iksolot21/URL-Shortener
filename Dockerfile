FROM golang:1.22-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o url-shortener ./cmd/url-shortener

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/url-shortener .
COPY config ./config

EXPOSE 8082

CMD ["./url-shortener"]