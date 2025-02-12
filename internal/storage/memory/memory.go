package memory

import (
	"sync"

	"url-shortener/internal/storage"
)

type MemoryStorage struct {
	mu      sync.RWMutex
	data    map[string]string // shortURL -> originalURL
	revData map[string]string // originalURL -> shortURL
}

func New() *MemoryStorage {
	return &MemoryStorage{
		data:    make(map[string]string),
		revData: make(map[string]string),
	}
}

func (s *MemoryStorage) SaveURL(originalURL string, shortURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.data[shortURL]; ok {
		return storage.ErrURLExists
	}
	if _, ok := s.revData[originalURL]; ok {
		return storage.ErrURLExists
	}

	s.data[shortURL] = originalURL
	s.revData[originalURL] = shortURL
	return nil
}

func (s *MemoryStorage) GetURL(shortURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	originalURL, ok := s.data[shortURL]
	if !ok {
		return "", storage.ErrURLNotFound
	}

	return originalURL, nil
}

func (s *MemoryStorage) GetShortURL(originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	shortURL, ok := s.revData[originalURL]
	if !ok {
		return "", storage.ErrURLNotFound
	}

	return shortURL, nil
}
