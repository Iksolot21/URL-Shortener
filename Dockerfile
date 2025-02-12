FROM golang:1.22-alpine

WORKDIR /app

COPY . .
RUN go mod download

RUN go build -o url-shortener ./cmd/url-shortener

EXPOSE 8082

CMD ["./url-shortener"]
