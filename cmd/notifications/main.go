package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	notificationsv1 "github.com/manovaspace/orbit-notifications/api/notifications/v1"
	"github.com/manovaspace/orbit-notifications/internal/application"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/clock"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/featureflags"
	grpchandlers "github.com/manovaspace/orbit-notifications/internal/infrastructure/grpc"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/httpapi"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/ids"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/postgres"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/postgres/migrations"
	"github.com/manovaspace/orbit-notifications/internal/infrastructure/smtp"
	observability "github.com/manovaspace/orbit-observability"
	googlegrpc "google.golang.org/grpc"
)

var version = "dev"

func main() {
	ctx := context.Background()
	if err := observability.Configure(observability.ConfigFromEnv("orbit-notifications", version)); err != nil {
		panic(err)
	}
	log := observability.Logger()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Error("DATABASE_URL is required")
		os.Exit(1)
	}

	if len(os.Args) > 1 && os.Args[1] == "migrate" {
		runMigrate(ctx, dsn, os.Args[2:])
		return
	}

	// Auto-migrate on start keeps the previous deployment behaviour. Set
	// MIGRATE_ON_START=false to require the explicit `migrate` subcommand.
	if !strings.EqualFold(envOr("MIGRATE_ON_START", "true"), "false") {
		if err := migrateUp(ctx, dsn); err != nil {
			log.Error("migrate failed", "error", err)
			os.Exit(1)
		}
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
	svc := application.NewService(application.Deps{
		Store:  repo,
		Mail:   mail,
		Flags:  flags,
		Clock:  clock.New(),
		IDs:    ids.New(),
		Dev:    os.Getenv("DEPLOYMENT_ENVIRONMENT") == "dev",
		Logger: func(msg string, args ...any) { log.Info(msg, args...) },
	})

	grpcPort := envOr("GRPC_PORT", "10110")
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Error("listen failed", "error", err)
		os.Exit(1)
	}

	gs := googlegrpc.NewServer(observability.GRPCServerOptions()...)
	notificationsv1.RegisterNotificationsServiceServer(gs, grpchandlers.New(svc))

	health := httpapi.New()
	healthPort := envOr("HEALTH_PORT", "10111")

	go func() {
		log.Info("grpc listening", "port", grpcPort)
		if err := gs.Serve(lis); err != nil {
			log.Error("grpc serve failed", "error", err)
		}
	}()

	go func() {
		log.Info("http listening", "port", healthPort)
		if err := health.Listen(":" + healthPort); err != nil {
			log.Error("http server failed", "error", err)
		}
	}()

	observability.WaitForSignal(observability.ShutdownConfig{
		GRPCServer: gs,
		OnShutdown: []func(context.Context) error{
			func(ctx context.Context) error { return health.Shutdown(ctx) },
			func(context.Context) error {
				pool.Close()
				return nil
			},
		},
	})
}

// runMigrate executes the explicit migrate subcommand:
//
//	notifications migrate          apply all pending migrations
//	notifications migrate up       apply all pending migrations
//	notifications migrate down     roll back the most recent migration
//	notifications migrate status   print every embedded migration
func runMigrate(ctx context.Context, dsn string, args []string) {
	action := "up"
	if len(args) > 0 {
		action = args[0]
	}
	var err error
	switch action {
	case "up":
		err = migrateUp(ctx, dsn)
	case "down":
		err = migrateDown(ctx, dsn)
	case "status":
		err = migrateStatus(ctx, dsn)
	default:
		err = fmt.Errorf("unknown migrate action %q (want up, down or status)", action)
	}
	if err != nil {
		observability.Logger().Error("migrate failed", "error", err)
		os.Exit(1)
	}
}

func migrateUp(ctx context.Context, dsn string) error {
	m, err := migrations.Open(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	results, err := m.Up(ctx)
	if err != nil {
		return err
	}
	for _, result := range results {
		observability.Logger().Info("migration applied",
			"version", result.Source.Version, "file", result.Source.Path)
	}
	return nil
}

func migrateDown(ctx context.Context, dsn string) error {
	m, err := migrations.Open(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	result, err := m.Down(ctx)
	if err != nil {
		return err
	}
	observability.Logger().Info("migration rolled back",
		"version", result.Source.Version, "file", result.Source.Path)
	return nil
}

func migrateStatus(ctx context.Context, dsn string) error {
	m, err := migrations.Open(dsn)
	if err != nil {
		return err
	}
	defer func() { _ = m.Close() }()

	statuses, err := m.Status(ctx)
	if err != nil {
		return err
	}
	for _, status := range statuses {
		observability.Logger().Info("migration",
			"version", status.Source.Version,
			"file", status.Source.Path,
			"applied", status.State == "applied")
	}
	current, target, err := m.Version(ctx)
	if err != nil {
		return err
	}
	observability.Logger().Info("migration version", "current", current, "latest", target)
	return nil
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
