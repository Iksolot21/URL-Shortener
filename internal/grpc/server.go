package grpc

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"
)

const shortURLLength = 10

type urlShortenerServer struct {
	storage storage.URLSaverURLGetter
	UnimplementedURLShortenerServer
	mu      sync.Mutex
	urlMap  map[string]string
	randGen *rand.Rand
}

func NewURLShortenerServer(storage storage.URLSaverURLGetter) *urlShortenerServer {
	return &urlShortenerServer{
		storage:                         storage,
		UnimplementedURLShortenerServer: UnimplementedURLShortenerServer{},
		mu:                              sync.Mutex{},
		urlMap:                          make(map[string]string),
		randGen:                         rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func (s *urlShortenerServer) CreateShortURL(ctx context.Context, req *CreateShortURLRequest) (*CreateShortURLResponse, error) {
	originalURL := req.OriginalUrl
	customAlias := req.CustomAlias

	if originalURL == "" {
		return nil, status.Error(codes.InvalidArgument, "original_url is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	var shortURL string

	if customAlias != "" {
		_, err := s.storage.GetURL(customAlias)
		if err == nil {
			return nil, status.Error(codes.AlreadyExists, "custom alias already exists")
		}

		if !errors.Is(err, storage.ErrURLNotFound) {
			log.Printf("failed to get URL: %v", err)
			return nil, status.Error(codes.Internal, "internal error")
		}

		shortURL = customAlias
	} else {
		shortURL = s.generateUniqueShortURL()
	}

	err := s.storage.SaveURL(originalURL, shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			return nil, status.Error(codes.AlreadyExists, "url already exists")
		}
		log.Printf("failed to save url: %v", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	return &CreateShortURLResponse{ShortUrl: shortURL}, nil
}

func (s *urlShortenerServer) GetOriginalURL(ctx context.Context, req *GetOriginalURLRequest) (*GetOriginalURLResponse, error) {
	shortURL := req.ShortUrl
	if shortURL == "" {
		return nil, status.Error(codes.InvalidArgument, "short_url is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	originalURL, ok := s.urlMap[shortURL]
	if !ok {
		var err error
		originalURL, err = s.storage.GetURL(shortURL)
		if err != nil {
			if errors.Is(err, storage.ErrURLNotFound) {
				return nil, status.Error(codes.NotFound, "short_url not found")
			}
			log.Printf("failed to get url: %v", err)
			return nil, status.Error(codes.Internal, "internal error")
		}
		s.urlMap[shortURL] = originalURL
	}

	return &GetOriginalURLResponse{OriginalUrl: originalURL}, nil
}

func (s *urlShortenerServer) generateUniqueShortURL() string {
	const maxAttempts = 10

	var shortURL string
	for attempt := 0; attempt < maxAttempts; attempt++ {
		shortURL = random.NewRandomString(shortURLLength)

		_, err := s.storage.GetURL(shortURL)
		if errors.Is(err, storage.ErrURLNotFound) {
			return shortURL
		}

		if err != nil {
			log.Printf("failed to get URL: %v", err)
			continue
		}
	}

	panic("failed to generate unique short URL")
}

func (s *urlShortenerServer) mustEmbedUnimplementedURLShortenerServer() {}

func StartGRPCServer(grpcAddress string, urlStorage storage.URLSaverURLGetter) error {
	lis, err := net.Listen("tcp", grpcAddress)
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s := grpc.NewServer()
	srv := NewURLShortenerServer(urlStorage)
	RegisterURLShortenerServer(s, srv)

	log.Printf("gRPC server listening on %s", grpcAddress)

	if err := s.Serve(lis); err != nil {
		return fmt.Errorf("failed to serve: %w", err)
	}
	return nil
}
