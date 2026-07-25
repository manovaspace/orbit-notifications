package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"time"

	observability "github.com/manovaspace/orbit-observability"
	notificationsv1 "github.com/manovaspace/orbit-notifications/api/notifications/v1"
	"github.com/manovaspace/orbit-notifications/internal/application"
	grpchandlers "github.com/manovaspace/orbit-notifications/internal/infrastructure/grpc"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/featureflags"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/postgres"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/smtp"
	"github.com/jackc/pgx/v5/pgxpool"
	googlegrpc "google.golang.org/grpc"
)

func main() {
	ctx := context.Background()
	if err := observability.Configure(observability.ConfigFromEnv("orbit-notifications", "0.1.0")); err != nil {
		panic(err)
	}
	log := observability.Logger()

	dsn := envOr("DATABASE_URL", "postgres://orbit:orbit@localhost:10332/notifications?sslmode=disable")
	if err := postgres.Migrate(ctx, dsn, "migrations"); err != nil {
		log.Error("migrate failed", "error", err)
		os.Exit(1)
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Error("db connect failed", "error", err)
		os.Exit(1)
	}

	repo := postgres.NewRepository(pool)
	mail, err := smtp.NewFromEnv()
	if err != nil {
		log.Error("smtp config failed", "error", err)
		os.Exit(1)
	}
	flags := featureflags.NewFromEnv("orbit-notifications", func(msg string, args ...any) {
		log.Info(msg, args...)
	})
	svc := application.NewService(repo, mail, flags, func(msg string, args ...any) {
		log.Info(msg, args...)
	})

	grpcPort := envOr("GRPC_PORT", "10110")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Error("listen failed", "error", err)
		os.Exit(1)
	}

	gs := googlegrpc.NewServer(observability.GRPCServerOptions()...)
	notificationsv1.RegisterNotificationsServiceServer(gs, grpchandlers.New(svc))

	healthMux := http.NewServeMux()
	healthMux.Handle("/healthz", observability.HealthHandler())
	healthMux.Handle("/readyz", observability.ReadinessHandler())
	healthPort := envOr("HEALTH_PORT", "10111")
	healthServer := &http.Server{
		Addr:              ":" + healthPort,
		Handler:           observability.HTTPMiddleware(healthMux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("grpc listening", "port", grpcPort)
		if err := gs.Serve(lis); err != nil {
			log.Error("grpc serve failed", "error", err)
		}
	}()

	go func() {
		if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("health server failed", "error", err)
		}
	}()

	observability.WaitForSignal(observability.ShutdownConfig{
		GRPCServer: gs,
		HTTPServer: healthServer,
		OnShutdown: []func(context.Context) error{
			func(context.Context) error {
				pool.Close()
				return nil
			},
		},
	})
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
