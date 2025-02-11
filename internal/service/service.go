package service

import (
	"context"
	"errors"
	"log"

	"url-shortener/internal/lib/random"
	"url-shortener/internal/storage"
)

var (
	ErrURLNotFound        = errors.New("url not found")
	ErrURLExists          = errors.New("url already exists")
	ErrAliasAlreadyExists = errors.New("custom alias already exists") // New error
	ErrInternal           = errors.New("internal error")
)

type URLShortenerService struct {
	storage        storage.URLSaverURLGetter
	shortURLLength int
}

func NewURLShortenerService(storage storage.URLSaverURLGetter, shortURLLength int) *URLShortenerService {
	return &URLShortenerService{
		storage:        storage,
		shortURLLength: shortURLLength,
	}
}

func (s *URLShortenerService) CreateShortURL(ctx context.Context, originalURL string, customAlias string) (string, error) {
	if originalURL == "" {
		return "", errors.New("original_url is required")
	}

	var shortURL string

	if customAlias != "" {
		// User provided a custom alias
		// Check if the alias is already taken
		_, err := s.storage.GetURL(customAlias)
		if err == nil {
			return "", ErrAliasAlreadyExists // Custom alias already exists
		}

		if !errors.Is(err, storage.ErrURLNotFound) {
			log.Printf("failed to get URL: %v", err)
			return "", ErrInternal // Internal error
		}

		shortURL = customAlias
	} else {
		// Generate a random short URL
		shortURL = s.generateUniqueShortURL()
	}

	err := s.storage.SaveURL(originalURL, shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			return "", ErrURLExists // URL already exists
		}
		log.Printf("failed to save url: %v", err)
		return "", ErrInternal // Internal error
	}

	return shortURL, nil
}

func (s *URLShortenerService) GetOriginalURL(ctx context.Context, shortURL string) (string, error) {
	if shortURL == "" {
		return "", errors.New("short_url is required")
	}

	originalURL, err := s.storage.GetURL(shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			return "", errors.New("short_url not found")
		}
		log.Printf("failed to get url: %v", err)
		return "", errors.New("internal error")
	}

	return originalURL, nil
}

func (s *URLShortenerService) generateUniqueShortURL() string {
	const maxAttempts = 10 // Prevent infinite loops

	for attempt := 0; attempt < maxAttempts; attempt++ {
		shortURL := random.NewRandomString(s.shortURLLength)

		// Check if short URL already exists
		_, err := s.storage.GetURL(shortURL)
		if errors.Is(err, storage.ErrURLNotFound) {
			return shortURL // Found a unique URL
		}

		if err != nil {
			log.Printf("failed to get URL: %v", err)
			continue // Try again
		}

	}

	panic("failed to generate unique short URL")
}
