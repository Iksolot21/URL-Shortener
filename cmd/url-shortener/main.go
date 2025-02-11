package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"golang.org/x/exp/slog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"url-shortener/internal/config"
	mygrpc "url-shortener/internal/grpc"
	"url-shortener/internal/lib/logger/handlers/slogpretty"
	"url-shortener/internal/lib/logger/sl"
	"url-shortener/internal/service"
	"url-shortener/internal/storage"
	"url-shortener/internal/storage/memory"
	"url-shortener/internal/storage/postgres"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("no .env file found")
	}

	cfg := config.MustLoad()

	slogLogger := setupLogger(cfg.Env)

	slogLogger.Info(
		"starting url-shortener",
		slog.String("env", cfg.Env),
		slog.String("version", "123"),
	)
	slogLogger.Debug("debug messages are enabled")

	var urlStorage storage.URLSaverURLGetter
	switch cfg.StorageType {
	case "memory":
		slogLogger.Info("using in-memory storage")
		urlStorage = memory.New()
	case "postgres":
		slogLogger.Info("using postgres storage")
		dataSourceName := os.Getenv("DATABASE_URL")
		if dataSourceName == "" {
			slogLogger.Error("DATABASE_URL is not set")
			os.Exit(1)
		}

		postgresStorage, err := postgres.New(dataSourceName)
		if err != nil {
			slogLogger.Error("failed to init postgres storage", sl.Err(err))
			os.Exit(1)
		}
		urlStorage = postgresStorage
	default:
		slogLogger.Error("invalid storage type", slog.String("storage_type", cfg.StorageType))
		os.Exit(1)
	}

	// Initialize service
	urlShortenerService := service.NewURLShortenerService(urlStorage, cfg.ShortURLLength)

	grpcServer := grpc.NewServer()
	urlShortenerServer := mygrpc.NewURLShortenerServer(urlShortenerService)
	mygrpc.RegisterURLShortenerServer(grpcServer, urlShortenerServer)
	reflection.Register(grpcServer)

	lis, err := net.Listen("tcp", cfg.HTTPServer.Address)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	log.Printf("gRPC server listening on %s", cfg.HTTPServer.Address)

	go func() {
		if err := grpcServer.Serve(lis); err != nil {
			log.Fatalf("failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slogLogger.Info("Gracefully shutting down gRPC server...")
	grpcServer.GracefulStop()
	slogLogger.Info("gRPC server stopped")

	fmt.Println("gRPC  server is closing")

}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}),
		)
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	default:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}),
		)
	}

	return log
}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}
