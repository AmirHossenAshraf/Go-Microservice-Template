package main

import (
	"Go-Microservice-Template/internal/config"
	"Go-Microservice-Template/internal/handler"
	"Go-Microservice-Template/internal/repository"
	"Go-Microservice-Template/internal/service"
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load config: %v\n", err)
		os.Exit(1)
	}

	// Setup structured logging
	setupLogger(cfg.LogLevel)
	log.Info().Str("version", cfg.Version).Msg("starting microservice")

	// Initialize dependencies
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Database connection
	db, err := repository.NewPostgresPool(ctx, cfg.DatabaseURL())
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to database")
	}

	// Redis connection
	cache, err := repository.NewRedisClient(ctx, cfg.RedisURL())
	if err != nil {
		log.Warn().Err(err).Msg("failed to connect to Redis, continuing without cache")
	} else {
		defer cache.Close()
		log.Info().Msg("connected to Redis")
	}
	// Build layers (Dependency Injection)
	userRepo := repository.NewUserRepository(db)
	userCache := repository.NewUserCache(cache, 5*time.Minute)
	userService := service.NewUserService(userRepo, userCache)
	httpHandler := handler.NewHTTPHandler(userService)
	grpcHandler := handler.NewGRPCHandler(userService)

	// ── HTTP Server ──────────────────────────────────────
	router := setupHTTPRouter(cfg, httpHandler)
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.HTTPPort),
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// ── gRPC Server ──────────────────────────────────────
	grpcServer := setupGRPCServer(cfg, grpcHandler)

	// ── Start servers ────────────────────────────────────
	errChan := make(chan error, 2)

	// Start HTTP
	go func() {
		log.Info().Int("port", cfg.HTTPPort).Msg("HTTP server starting")
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- fmt.Errorf("HTTP server error: %w", err)
		}
	}()

	// Start gRPC
	go func() {
		lis, err := net.Listen("tcp", fmt.Sprintf(":%d", cfg.GRPCPort))
		if err != nil {
			errChan <- fmt.Errorf("gRPC listen error: %w", err)
			return
		}
		log.Info().Int("port", cfg.GRPCPort).Msg("gRPC server starting")
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- fmt.Errorf("gRPC server error: %w", err)
		}
	}()

	// ── Graceful Shutdown ────────────────────────────────
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		log.Info().Str("signal", sig.String()).Msg("shutting down gracefully")
	case err := <-errChan:
		log.Error().Err(err).Msg("server error, shutting down")
	}

	// Give active connections 30 seconds to finish
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// Shutdown HTTP
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("HTTP server forced shutdown")
	}

	// Shutdown gRPC
	grpcServer.GracefulStop()

	log.Info().Msg("server stopped cleanly")
}

func setupLogger(level string) {
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	lvl, err := zerolog.ParseLevel(level)
	if err != nil {
		lvl = zerolog.InfoLevel
	}
	zerolog.SetGlobalLevel(lvl)

	// Pretty logging for development, JSON for production
	if os.Getenv("APP_ENV") != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}
}

func setupHTTPRouter(cfg *config.Config, h *handler.HTTPHandler) *chi.Mux {
	r := chi.NewRouter()

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// Health & metrics (public)
	r.Get("/health", h.Health)
	r.Get("/readiness", h.Readiness)
	r.Get("/metrics", h.Metrics)

	return r
}

func setupGRPCServer(cfg *config.Config, h *handler.GRPCHandler) *grpc.Server {
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(4 * 1024 * 1024), // 4MB

	}

	server := grpc.NewServer(opts...)

	// Register services
	h.Register(server)

	// Enable reflection for debugging
	reflection.Register(server)

	return server
}
