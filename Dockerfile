FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o url-shortener ./cmd/url-shortener

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/url-shortener .
COPY config ./config

# Set environment variables
ENV STORAGE_TYPE=memory
#ENV DATABASE_URL="postgres://user:password@host:port/database?sslmode=disable"

EXPOSE 8082

CMD ["./url-shortener"]