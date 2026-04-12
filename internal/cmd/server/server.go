package server

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strconv"
	"strings"

	gamev1 "github.com/fracturing-space/game/api/gen/go/game/v1"
	"github.com/fracturing-space/game/internal/service"
	sqlitestorage "github.com/fracturing-space/game/internal/storage/sqlite"
	gamev1transport "github.com/fracturing-space/game/internal/transport/grpc/gamev1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/health"
	healthpb "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

const (
	defaultPort          = 8080
	envPort              = "FRACTURING_SPACE_GAME_PORT"
	envAddr              = "FRACTURING_SPACE_GAME_ADDR"
	envEventsDBPath      = "FRACTURING_SPACE_GAME_EVENTS_DB_PATH"
	envProjectionsDBPath = "FRACTURING_SPACE_GAME_PROJECTIONS_DB_PATH"
	envArtifactsDBPath   = "FRACTURING_SPACE_GAME_ARTIFACTS_DB_PATH"
)

// Config holds server command configuration.
type Config struct {
	Port              int
	Addr              string
	EventsDBPath      string
	ProjectionsDBPath string
	ArtifactsDBPath   string
}

// ParseConfig parses environment defaults and command-line flags into a Config.
func ParseConfig(fs *flag.FlagSet, args []string) (Config, error) {
	if fs == nil {
		return Config{}, fmt.Errorf("flag set is required")
	}
	fs.SetOutput(os.Stderr)

	cfg, err := loadConfigFromEnv()
	if err != nil {
		return Config{}, err
	}

	fs.IntVar(&cfg.Port, "port", cfg.Port, "game server port")
	fs.StringVar(&cfg.Addr, "addr", cfg.Addr, "game server listen address (overrides -port)")
	fs.StringVar(&cfg.EventsDBPath, "events-db-path", cfg.EventsDBPath, "path to events sqlite database")
	fs.StringVar(&cfg.ProjectionsDBPath, "projections-db-path", cfg.ProjectionsDBPath, "path to projections sqlite database")
	fs.StringVar(&cfg.ArtifactsDBPath, "artifacts-db-path", cfg.ArtifactsDBPath, "path to artifacts sqlite database")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if fs.NArg() != 0 {
		return Config{}, fmt.Errorf("unexpected positional arguments: %v", fs.Args())
	}
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func loadConfigFromEnv() (Config, error) {
	cfg := Config{
		Port:              defaultPort,
		Addr:              strings.TrimSpace(os.Getenv(envAddr)),
		EventsDBPath:      strings.TrimSpace(os.Getenv(envEventsDBPath)),
		ProjectionsDBPath: strings.TrimSpace(os.Getenv(envProjectionsDBPath)),
		ArtifactsDBPath:   strings.TrimSpace(os.Getenv(envArtifactsDBPath)),
	}
	defaultPaths := sqlitestorage.DefaultPaths()
	if cfg.EventsDBPath == "" {
		cfg.EventsDBPath = defaultPaths.EventsDBPath
	}
	if cfg.ProjectionsDBPath == "" {
		cfg.ProjectionsDBPath = defaultPaths.ProjectionsDBPath
	}
	if cfg.ArtifactsDBPath == "" {
		cfg.ArtifactsDBPath = defaultPaths.ArtifactsDBPath
	}

	rawPort := strings.TrimSpace(os.Getenv(envPort))
	if rawPort == "" {
		return cfg, nil
	}
	port, err := strconv.Atoi(rawPort)
	if err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", envPort, err)
	}
	cfg.Port = port
	if err := validateConfig(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validateConfig(cfg Config) error {
	if cfg.Port < 0 || cfg.Port > 65535 {
		return fmt.Errorf("port must be between 0 and 65535")
	}
	return nil
}

func effectiveListenAddr(cfg Config) string {
	if addr := strings.TrimSpace(cfg.Addr); addr != "" {
		return addr
	}
	return fmt.Sprintf(":%d", cfg.Port)
}

// Run starts the local game gRPC server and blocks until it stops or ctx ends.
func Run(ctx context.Context, cfg Config) error {
	if ctx == nil {
		ctx = context.Background()
	}

	manifest, err := service.BuildManifest(nil)
	if err != nil {
		return fmt.Errorf("build manifest: %w", err)
	}
	stores, err := sqlitestorage.Open(manifest, sqlitestorage.Paths{
		EventsDBPath:      cfg.EventsDBPath,
		ProjectionsDBPath: cfg.ProjectionsDBPath,
		ArtifactsDBPath:   cfg.ArtifactsDBPath,
	})
	if err != nil {
		return fmt.Errorf("open sqlite stores: %w", err)
	}
	defer stores.Close()

	svc, err := service.New(service.Config{
		Manifest:        manifest,
		IDs:             service.NewOpaqueIDAllocator(),
		Journal:         stores.Journal,
		ProjectionStore: stores.ProjectionStore,
		ArtifactStore:   stores.ArtifactStore,
		Logger:          slog.Default(),
	})
	if err != nil {
		return fmt.Errorf("build service: %w", err)
	}
	handler, err := gamev1transport.NewServer(svc)
	if err != nil {
		return fmt.Errorf("build transport: %w", err)
	}

	listener, err := net.Listen("tcp", effectiveListenAddr(cfg))
	if err != nil {
		return fmt.Errorf("listen: %w", err)
	}
	defer listener.Close()

	server := grpc.NewServer()
	gamev1.RegisterGameServiceServer(server, handler)

	healthServer := health.NewServer()
	healthServer.SetServingStatus("", healthpb.HealthCheckResponse_SERVING)
	healthServer.SetServingStatus("game.v1.GameService", healthpb.HealthCheckResponse_SERVING)
	healthpb.RegisterHealthServer(server, healthServer)

	reflection.Register(server)

	slog.Info("game server listening", "addr", listener.Addr().String())

	return runServeLoop(
		ctx,
		func() error {
			return server.Serve(listener)
		},
		func() {
			healthServer.SetServingStatus("", healthpb.HealthCheckResponse_NOT_SERVING)
			healthServer.SetServingStatus("game.v1.GameService", healthpb.HealthCheckResponse_NOT_SERVING)
			healthServer.Shutdown()
			server.GracefulStop()
		},
	)
}

func normalizeServeError(err error) error {
	if err == nil || errors.Is(err, grpc.ErrServerStopped) {
		return nil
	}
	return fmt.Errorf("serve gRPC: %w", err)
}

func runServeLoop(ctx context.Context, serveFn func() error, shutdownFn func()) error {
	if serveFn == nil {
		return fmt.Errorf("serve function is required")
	}
	if shutdownFn == nil {
		shutdownFn = func() {}
	}

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- serveFn()
	}()

	select {
	case <-ctx.Done():
		shutdownFn()
		return normalizeServeError(<-serveErr)
	case err := <-serveErr:
		return normalizeServeError(err)
	}
}
