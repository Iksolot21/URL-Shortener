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
	ErrAliasAlreadyExists = errors.New("custom alias already exists")
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

	shortURL, err := s.getExistingShortURL(ctx, originalURL)
	if err == nil {
		return shortURL, nil
	}
	if !errors.Is(err, ErrURLNotFound) {
		log.Printf("failed to get existing short URL: %v", err)
		return "", ErrInternal
	}

	if customAlias != "" {
		_, err := s.storage.GetURL(customAlias)
		if err == nil {
			return "", ErrAliasAlreadyExists
		}

		if !errors.Is(err, storage.ErrURLNotFound) {
			log.Printf("failed to get URL: %v", err)
			return "", ErrInternal
		}

		shortURL = customAlias
	} else {
		shortURL = s.generateUniqueShortURL()
	}

	err = s.storage.SaveURL(originalURL, shortURL)
	if err != nil {
		if errors.Is(err, storage.ErrURLExists) {
			shortURL, err := s.getExistingShortURL(ctx, originalURL)
			if err == nil {
				return shortURL, nil
			}
			log.Printf("failed to get existing short URL: %v", err)
			return "", ErrInternal
		}
		log.Printf("failed to save url: %v", err)
		return "", ErrInternal
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
			return "", ErrURLNotFound
		}
		log.Printf("failed to get url: %v", err)
		return "", ErrInternal
	}

	return originalURL, nil
}

func (s *URLShortenerService) generateUniqueShortURL() string {
	const maxAttempts = 10

	for attempt := 0; attempt < maxAttempts; attempt++ {
		shortURL := random.NewRandomString(s.shortURLLength)

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
