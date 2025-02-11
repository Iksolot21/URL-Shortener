package tests

import (
	"context"
	"log"
	"net"
	"os"
	"testing"
	"time"

	"github.com/joho/godotenv"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	mygrpc "url-shortener/internal/grpc"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
	"url-shortener/internal/storage/memory"
	"url-shortener/internal/storage/postgres"
)

const bufSize = 1024 * 1024

const (
	testDBConnectionStringEnv = "TEST_DATABASE_URL"
)

func newTestPostgresStorage(t *testing.T) *postgres.PostgresStorage {
	t.Helper()
	connectionString := os.Getenv(testDBConnectionStringEnv)
	if connectionString == "" {
		t.Fatalf("must set %s env var", testDBConnectionStringEnv)
	}

	log.Printf("TEST_DATABASE_URL: %s", connectionString)

	pgStorage, err := postgres.New(connectionString)
	if err != nil {
		t.Fatalf("failed to create postgres storage: %v", err)
	}

	if err := cleanDatabase(pgStorage); err != nil {
		t.Fatalf("failed to clean database: %v", err)
	}

	return pgStorage
}

func cleanDatabase(pgStorage *postgres.PostgresStorage) error {
	_, err := pgStorage.Db.Exec("DELETE FROM urls")
	return err
}

func newBufConnListener(t *testing.T, server *grpc.Server) *bufconn.Listener {
	lis := bufconn.Listen(bufSize)
	go func() {
		if err := server.Serve(lis); err != nil {
			t.Fatalf("Server exited with error: %v", err)
		}
	}()
	return lis
}

func newTestClient(t *testing.T, lis *bufconn.Listener) (mygrpc.URLShortenerClient, func()) {
	conn, err := grpc.DialContext(context.Background(), "bufnet",
		grpc.WithDialer(func(string, time.Duration) (net.Conn, error) {
			return lis.Dial()
		}), grpc.WithInsecure())

	if err != nil {
		t.Fatalf("Failed to dial bufnet: %v", err)
	}
	close := func() {
		conn.Close()
	}
	client := mygrpc.NewURLShortenerClient(conn)
	return client, close
}

func newTestService(t *testing.T, storage storage.URLSaverURLGetter) *service.URLShortenerService {
	t.Helper()
	return service.NewURLShortenerService(storage, 10)
}

func TestMain(m *testing.M) {
	err := godotenv.Load("./.env")
	if err != nil {
		log.Printf("no .env file found")
	}
	os.Exit(m.Run())
}

func TestCreateShortURL_Postgres(t *testing.T) {

	pgStorage := newTestPostgresStorage(t)
	defer func() {
		if err := pgStorage.Close(); err != nil {
			t.Fatalf("failed to close database connection: %v", err)
		}
	}()

	s := grpc.NewServer()
	urlShortenerServer := mygrpc.NewURLShortenerServer(newTestService(t, pgStorage))
	mygrpc.RegisterURLShortenerServer(s, urlShortenerServer)

	lis := newBufConnListener(t, s)
	client, close := newTestClient(t, lis)
	defer close()
	defer s.GracefulStop()

	originalURL := "https://example.com"
	resp, err := client.CreateShortURL(context.Background(), &mygrpc.CreateShortURLRequest{OriginalUrl: originalURL})
	if err != nil {
		t.Fatalf("CreateShortURL failed: %v", err)
	}

	if resp.ShortUrl == "" {
		t.Errorf("ShortURL should not be empty")
	}

	url, err := pgStorage.GetURL(resp.ShortUrl)
	if err != nil {
		t.Fatalf("Failed to get URL from storage: %v", err)
	}

	if url != originalURL {
		t.Errorf("URL in storage is not the same as original URL")
	}
}

func TestGetOriginalURL_Postgres(t *testing.T) {
	pgStorage := newTestPostgresStorage(t)
	defer func() {
		if err := pgStorage.Close(); err != nil {
			t.Fatalf("failed to close database connection: %v", err)
		}
	}()

	s := grpc.NewServer()

	originalURL := "https://example.com"
	shortURL := "test"

	err := pgStorage.SaveURL(originalURL, shortURL)
	if err != nil {
		t.Fatalf("Failed to save url to database %v", err)
	}

	urlShortenerServer := mygrpc.NewURLShortenerServer(newTestService(t, pgStorage))
	mygrpc.RegisterURLShortenerServer(s, urlShortenerServer)

	lis := newBufConnListener(t, s)
	client, close := newTestClient(t, lis)
	defer close()
	defer s.GracefulStop()

	resp, err := client.GetOriginalURL(context.Background(), &mygrpc.GetOriginalURLRequest{ShortUrl: shortURL})

	if err != nil {
		t.Fatalf("GetOriginalURL failed: %v", err)
	}

	if resp.OriginalUrl != originalURL {
		t.Errorf("OriginalURL should be https://example.com, but got %s", resp.OriginalUrl)
	}
}

func TestCreateShortURL_CustomAliasAlreadyExists_Postgres(t *testing.T) {
	pgStorage := newTestPostgresStorage(t)
	defer func() {
		if err := pgStorage.Close(); err != nil {
			t.Fatalf("failed to close database connection: %v", err)
		}
	}()

	s := grpc.NewServer()

	originalURL := "https://example.com"
	shortURL := "existing"

	err := pgStorage.SaveURL(originalURL, shortURL)
	if err != nil {
		t.Fatalf("Failed to save url to database %v", err)
	}

	urlShortenerServer := mygrpc.NewURLShortenerServer(newTestService(t, pgStorage))
	mygrpc.RegisterURLShortenerServer(s, urlShortenerServer)

	lis := newBufConnListener(t, s)
	client, close := newTestClient(t, lis)
	defer close()
	defer s.GracefulStop()

	newOriginalURL := "https://example.org"
	customAlias := "existing"

	_, err = client.CreateShortURL(context.Background(), &mygrpc.CreateShortURLRequest{OriginalUrl: newOriginalURL, CustomAlias: customAlias})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected error to be a status error, got %v", err)
	}

	if st.Code() != codes.AlreadyExists {
		t.Errorf("Expected code to be %s, got %s", codes.AlreadyExists, st.Code())
	}

	if st.Message() != "custom alias already exists" {
		t.Errorf("Expected message to be 'custom alias already exists', got %s", st.Message())
	}
}

func TestCreateShortURL_InMemory(t *testing.T) {
	memStorage := memory.New()

	s := grpc.NewServer()
	urlShortenerServer := mygrpc.NewURLShortenerServer(newTestService(t, memStorage))
	mygrpc.RegisterURLShortenerServer(s, urlShortenerServer)

	lis := newBufConnListener(t, s)
	client, close := newTestClient(t, lis)
	defer close()
	defer s.GracefulStop()

	originalURL := "https://example.com"
	resp, err := client.CreateShortURL(context.Background(), &mygrpc.CreateShortURLRequest{OriginalUrl: originalURL})
	if err != nil {
		t.Fatalf("CreateShortURL failed: %v", err)
	}

	if resp.ShortUrl == "" {
		t.Errorf("ShortURL should not be empty")
	}

	url, err := memStorage.GetURL(resp.ShortUrl)
	if err != nil {
		t.Fatalf("Failed to get URL from storage: %v", err)
	}

	if url != originalURL {
		t.Errorf("URL in storage is not the same as original URL")
	}
}

func TestGetOriginalURL_InMemory(t *testing.T) {
	memStorage := memory.New()

	s := grpc.NewServer()

	originalURL := "https://example.com"
	shortURL := "test"

	err := memStorage.SaveURL(originalURL, shortURL)
	if err != nil {
		t.Fatalf("Failed to save url to memory storage %v", err)
	}

	urlShortenerServer := mygrpc.NewURLShortenerServer(newTestService(t, memStorage))
	mygrpc.RegisterURLShortenerServer(s, urlShortenerServer)

	lis := newBufConnListener(t, s)
	client, close := newTestClient(t, lis)
	defer close()
	defer s.GracefulStop()

	resp, err := client.GetOriginalURL(context.Background(), &mygrpc.GetOriginalURLRequest{ShortUrl: shortURL})

	if err != nil {
		t.Fatalf("GetOriginalURL failed: %v", err)
	}

	if resp.OriginalUrl != originalURL {
		t.Errorf("OriginalURL should be https://example.com, but got %s", resp.OriginalUrl)
	}
}

func TestCreateShortURL_CustomAliasAlreadyExists_InMemory(t *testing.T) {
	memStorage := memory.New()

	s := grpc.NewServer()

	originalURL := "https://example.com"
	shortURL := "existing"

	err := memStorage.SaveURL(originalURL, shortURL)
	if err != nil {
		t.Fatalf("Failed to save url to memory storage %v", err)
	}

	urlShortenerServer := mygrpc.NewURLShortenerServer(newTestService(t, memStorage))
	mygrpc.RegisterURLShortenerServer(s, urlShortenerServer)

	lis := newBufConnListener(t, s)
	client, close := newTestClient(t, lis)
	defer close()
	defer s.GracefulStop()

	newOriginalURL := "https://example.org"
	customAlias := "existing"

	_, err = client.CreateShortURL(context.Background(), &mygrpc.CreateShortURLRequest{OriginalUrl: newOriginalURL, CustomAlias: customAlias})
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("Expected error to be a status error, got %v", err)
	}

	if st.Code() != codes.AlreadyExists {
		t.Errorf("Expected code to be %s, got %s", codes.AlreadyExists, st.Code())
	}

	if st.Message() != "custom alias already exists" {
		t.Errorf("Expected message to be 'custom alias already exists', got %s", st.Message())
	}
}

type mockStorage struct {
	data map[string]string
}

func (m *mockStorage) SaveURL(urlToSave string, alias string) error {
	if _, ok := m.data[alias]; ok {
		return storage.ErrURLExists
	}

	m.data[alias] = urlToSave
	return nil
}

func (m *mockStorage) GetURL(alias string) (string, error) {
	url, ok := m.data[alias]
	if !ok {
		return "", storage.ErrURLNotFound
	}
	return url, nil
}
