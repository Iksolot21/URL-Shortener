package grpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"sync"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"url-shortener/internal/service"
	"url-shortener/internal/storage"
)

// urlShortenerServer реализует интерфейс URLShortenerServer, сгенерированный protoc
type urlShortenerServer struct {
	srv *service.URLShortenerService // Use pointer here
	UnimplementedURLShortenerServer
	mu     sync.Mutex
	urlMap map[string]string
}

func NewURLShortenerServer(srv *service.URLShortenerService) URLShortenerServer { // Accepts service interface
	return &urlShortenerServer{
		srv:                             srv,
		UnimplementedURLShortenerServer: UnimplementedURLShortenerServer{},
		mu:                              sync.Mutex{},
		urlMap:                          map[string]string{},
	}
}

func (s *urlShortenerServer) CreateShortURL(ctx context.Context, req *CreateShortURLRequest) (*CreateShortURLResponse, error) {
	originalURL := req.OriginalUrl
	customAlias := req.CustomAlias

	shortURL, err := s.srv.CreateShortURL(ctx, originalURL, customAlias) // Call service method
	if err != nil {
		log.Printf("failed to create short url: %v", err)
		if errors.Is(err, storage.ErrURLNotFound) {
			return nil, status.Error(codes.NotFound, "short_url not found")
		}
		if errors.Is(err, storage.ErrURLExists) {
			return nil, status.Error(codes.AlreadyExists, "url already exists")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &CreateShortURLResponse{ShortUrl: shortURL}, nil
}

func (s *urlShortenerServer) GetOriginalURL(ctx context.Context, req *GetOriginalURLRequest) (*GetOriginalURLResponse, error) {
	shortURL := req.ShortUrl

	originalURL, err := s.srv.GetOriginalURL(ctx, shortURL) // Call service method
	if err != nil {
		log.Printf("failed to get original url: %v", err)
		if errors.Is(err, storage.ErrURLNotFound) {
			return nil, status.Error(codes.NotFound, "short_url not found")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &GetOriginalURLResponse{OriginalUrl: originalURL}, nil
}

func (s *urlShortenerServer) mustEmbedUnimplementedURLShortenerServer() {}

// StartGRPCServer starts the gRPC server.
func StartGRPCServer(grpcAddress string, urlService *service.URLShortenerService) error { // Accepts service interface
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	srv := NewURLShortenerServer(urlService) // Create gRPC server with service
	RegisterURLShortenerServer(s, srv)

	log.Printf("gRPC server listening on %s", grpcAddress)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}
