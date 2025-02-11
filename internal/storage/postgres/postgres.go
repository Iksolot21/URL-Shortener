package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/lib/pq"

	"url-shortener/internal/storage"
)

type PostgresStorage struct {
	Db *sql.DB
	mu sync.Mutex
}

func New(dataSourceName string) (*PostgresStorage, error) {
	if dataSourceName == "" {
		dataSourceName = os.Getenv("DATABASE_URL")
		if dataSourceName == "" {
			return nil, errors.New("DATABASE_URL is not set")
		}
	}

	db, err := sql.Open("postgres", dataSourceName)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	err = db.PingContext(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	_, err = db.ExecContext(context.Background(), `
		CREATE TABLE IF NOT EXISTS urls (
			short_url TEXT PRIMARY KEY,
			original_url TEXT NOT NULL
		);
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return &PostgresStorage{Db: db}, nil
}

func (s *PostgresStorage) SaveURL(urlToSave string, alias string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	log.Printf("Saving URL: originalURL=%s, shortURL=%s", urlToSave, alias)

	_, err := s.Db.ExecContext(context.Background(),
		"INSERT INTO urls (short_url, original_url) VALUES ($1, $2)",
		alias, urlToSave,
	)
	if err != nil {
		log.Printf("Error inserting URL: %v", err)
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return storage.ErrURLExists
		}
		return fmt.Errorf("failed to insert url: %w", err)
	}

	return nil
}

func (s *PostgresStorage) GetURL(alias string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var originalURL string
	err := s.Db.QueryRowContext(context.Background(),
		"SELECT original_url FROM urls WHERE short_url = $1", alias).Scan(&originalURL)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrURLNotFound
		}
		return "", fmt.Errorf("failed to get url: %w", err)
	}

	return originalURL, nil
}

func (s *PostgresStorage) Close() error {
	return s.Db.Close()
}
