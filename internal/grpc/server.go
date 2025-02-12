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

type urlShortenerServer struct {
	srv *service.URLShortenerService
	UnimplementedURLShortenerServer
	mu     sync.Mutex
	urlMap map[string]string
}

func NewURLShortenerServer(srv *service.URLShortenerService) URLShortenerServer {
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

	shortURL, err := s.srv.CreateShortURL(ctx, originalURL, customAlias)
	if err != nil {
		log.Printf("failed to create short url: %v", err)
		if errors.Is(err, service.ErrURLNotFound) {
			return nil, status.Error(codes.NotFound, "short_url not found")
		}
		if errors.Is(err, service.ErrURLExists) {
			return nil, status.Error(codes.AlreadyExists, "url already exists")
		}
		if errors.Is(err, service.ErrAliasAlreadyExists) {
			return nil, status.Error(codes.AlreadyExists, "custom alias already exists")
		}
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &CreateShortURLResponse{ShortUrl: shortURL}, nil
}

func (s *urlShortenerServer) GetOriginalURL(ctx context.Context, req *GetOriginalURLRequest) (*GetOriginalURLResponse, error) {
	shortURL := req.ShortUrl

	originalURL, err := s.srv.GetOriginalURL(ctx, shortURL)
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

func StartGRPCServer(grpcAddress string, urlService *service.URLShortenerService) error {
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	srv := NewURLShortenerServer(urlService)
	RegisterURLShortenerServer(s, srv)

	log.Printf("gRPC server listening on %s", grpcAddress)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}
