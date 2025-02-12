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

	// Check if URL already exists
	shortURL, err := s.getExistingShortURL(ctx, originalURL)
	if err == nil {
		return shortURL, nil // Return existing short URL
	}
	if !errors.Is(err, ErrURLNotFound) {
		log.Printf("failed to get existing short URL: %v", err)
		return "", ErrInternal
	}

	if customAlias != "" {
		// User provided a custom alias
		// Check if the alias is already taken
		_, err := s.storage.GetURL(customAlias)
		if err == nil {
			return "", ErrAliasAlreadyExists //Custom alias already exists
		}

		if !errors.Is(err, storage.ErrURLNotFound) {
			log.Printf("failed to get URL: %v", err)
			return "", ErrInternal
		}

		shortURL = customAlias
	} else {
		// Generate a random short URL
		shortURL = s.generateUniqueShortURL()
	}

	err = s.storage.SaveURL(originalURL, shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			// Double check if short url exists. If exists, return shortUrl
			shortURL, err := s.getExistingShortURL(ctx, originalURL)
			if err == nil {
				return shortURL, nil // Return existing short URL
			}
			log.Printf("failed to get existing short URL: %v", err)
			return "", ErrInternal // Internal error
		}
		log.Printf("failed to save url: %v", err)
		return "", ErrInternal //Internal error
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
			return "", ErrURLNotFound // URL not found
		}
		log.Printf("failed to get url: %v", err)
		return "", ErrInternal // Internal error
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

		// URL already exists, generate another one
	}

	// If we reach here, it means we couldn't generate a unique URL after maxAttempts
	panic("failed to generate unique short URL")
}

func (s *URLShortenerService) getExistingShortURL(ctx context.Context, originalURL string) (string, error) {
	shortURL, err := s.storage.GetShortURL(originalURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLNotFound) {
			return "", ErrURLNotFound
		}
		log.Printf("failed to get short URL: %v", err)
		return "", ErrInternal
	}

	return shortURL, nil
}
