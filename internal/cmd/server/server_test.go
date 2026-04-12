package server

import (
	"context"
	"errors"
	"flag"
	"strings"
	"testing"

	sqlitestorage "github.com/fracturing-space/game/internal/storage/sqlite"
	"google.golang.org/grpc"
)

func TestParseConfigDefaults(t *testing.T) {
	t.Setenv(envPort, "")
	t.Setenv(envAddr, "")
	t.Setenv(envEventsDBPath, "")
	t.Setenv(envProjectionsDBPath, "")
	t.Setenv(envArtifactsDBPath, "")

	cfg, err := ParseConfig(flag.NewFlagSet("server", flag.ContinueOnError), nil)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if got, want := cfg.Port, defaultPort; got != want {
		t.Fatalf("port = %d, want %d", got, want)
	}
	if got := cfg.Addr; got != "" {
		t.Fatalf("addr = %q, want empty", got)
	}
	defaults := sqlitestorage.DefaultPaths()
	if got, want := cfg.EventsDBPath, defaults.EventsDBPath; got != want {
		t.Fatalf("events db path = %q, want %q", got, want)
	}
	if got, want := cfg.ProjectionsDBPath, defaults.ProjectionsDBPath; got != want {
		t.Fatalf("projections db path = %q, want %q", got, want)
	}
	if got, want := cfg.ArtifactsDBPath, defaults.ArtifactsDBPath; got != want {
		t.Fatalf("artifacts db path = %q, want %q", got, want)
	}
}

func TestParseConfigReadsEnv(t *testing.T) {
	t.Setenv(envPort, "9001")
	t.Setenv(envAddr, "127.0.0.1:9999")
	t.Setenv(envEventsDBPath, "/tmp/events.db")
	t.Setenv(envProjectionsDBPath, "/tmp/projections.db")
	t.Setenv(envArtifactsDBPath, "/tmp/artifacts.db")

	cfg, err := ParseConfig(flag.NewFlagSet("server", flag.ContinueOnError), nil)
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if got, want := cfg.Port, 9001; got != want {
		t.Fatalf("port = %d, want %d", got, want)
	}
	if got, want := cfg.Addr, "127.0.0.1:9999"; got != want {
		t.Fatalf("addr = %q, want %q", got, want)
	}
	if got, want := cfg.EventsDBPath, "/tmp/events.db"; got != want {
		t.Fatalf("events db path = %q, want %q", got, want)
	}
	if got, want := cfg.ProjectionsDBPath, "/tmp/projections.db"; got != want {
		t.Fatalf("projections db path = %q, want %q", got, want)
	}
	if got, want := cfg.ArtifactsDBPath, "/tmp/artifacts.db"; got != want {
		t.Fatalf("artifacts db path = %q, want %q", got, want)
	}
}

func TestParseConfigFlagsOverrideEnv(t *testing.T) {
	t.Setenv(envPort, "9001")
	t.Setenv(envAddr, "127.0.0.1:9999")
	t.Setenv(envEventsDBPath, "/tmp/events.db")
	t.Setenv(envProjectionsDBPath, "/tmp/projections.db")
	t.Setenv(envArtifactsDBPath, "/tmp/artifacts.db")

	cfg, err := ParseConfig(flag.NewFlagSet("server", flag.ContinueOnError), []string{
		"-port", "8088",
		"-addr", "127.0.0.1:8087",
		"-events-db-path", "/var/events.db",
		"-projections-db-path", "/var/projections.db",
		"-artifacts-db-path", "/var/artifacts.db",
	})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if got, want := cfg.Port, 8088; got != want {
		t.Fatalf("port = %d, want %d", got, want)
	}
	if got, want := cfg.Addr, "127.0.0.1:8087"; got != want {
		t.Fatalf("addr = %q, want %q", got, want)
	}
	if got, want := cfg.EventsDBPath, "/var/events.db"; got != want {
		t.Fatalf("events db path = %q, want %q", got, want)
	}
	if got, want := cfg.ProjectionsDBPath, "/var/projections.db"; got != want {
		t.Fatalf("projections db path = %q, want %q", got, want)
	}
	if got, want := cfg.ArtifactsDBPath, "/var/artifacts.db"; got != want {
		t.Fatalf("artifacts db path = %q, want %q", got, want)
	}
}

func TestParseConfigAddrOverridesPort(t *testing.T) {
	cfg, err := ParseConfig(flag.NewFlagSet("server", flag.ContinueOnError), []string{"-port", "8088", "-addr", "127.0.0.1:8087"})
	if err != nil {
		t.Fatalf("ParseConfig() error = %v", err)
	}
	if got, want := effectiveListenAddr(cfg), "127.0.0.1:8087"; got != want {
		t.Fatalf("effectiveListenAddr() = %q, want %q", got, want)
	}
}

func TestParseConfigRejectsPositionalArgs(t *testing.T) {
	_, err := ParseConfig(flag.NewFlagSet("server", flag.ContinueOnError), []string{"extra"})
	if err == nil {
		t.Fatal("ParseConfig() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "unexpected positional arguments") {
		t.Fatalf("ParseConfig() error = %v, want positional argument context", err)
	}
}

func TestParseConfigRejectsInvalidEnvPort(t *testing.T) {
	t.Setenv(envPort, "bad")

	_, err := ParseConfig(flag.NewFlagSet("server", flag.ContinueOnError), nil)
	if err == nil {
		t.Fatal("ParseConfig() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "parse "+envPort) {
		t.Fatalf("ParseConfig() error = %v, want env parse context", err)
	}
}

func TestParseConfigRejectsOutOfRangePort(t *testing.T) {
	t.Setenv(envPort, "70000")

	_, err := ParseConfig(flag.NewFlagSet("server", flag.ContinueOnError), nil)
	if err == nil {
		t.Fatal("ParseConfig() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "port must be between 0 and 65535") {
		t.Fatalf("ParseConfig() error = %v, want port range failure", err)
	}
}

func TestRunServeLoopRejectsMissingServeFn(t *testing.T) {
	if err := runServeLoop(context.Background(), nil, nil); err == nil {
		t.Fatal("runServeLoop() error = nil, want failure")
	}
}

func TestRunServeLoopContextCancelTriggersShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownCalled := false
	serveCalled := make(chan struct{}, 1)
	shutdownDone := make(chan struct{})

	errCh := make(chan error, 1)
	go func() {
		errCh <- runServeLoop(ctx, func() error {
			serveCalled <- struct{}{}
			<-shutdownDone
			return grpc.ErrServerStopped
		}, func() {
			shutdownCalled = true
			close(shutdownDone)
		})
	}()

	<-serveCalled
	cancel()

	err := <-errCh
	if err != nil {
		t.Fatalf("runServeLoop() error = %v", err)
	}
	if !shutdownCalled {
		t.Fatal("shutdown function was not called")
	}
}

func TestRunServeLoopReturnsServeError(t *testing.T) {
	serveErr := errors.New("boom")

	err := runServeLoop(context.Background(), func() error {
		return serveErr
	}, nil)
	if err == nil {
		t.Fatal("runServeLoop() error = nil, want failure")
	}
	if !strings.Contains(err.Error(), "serve gRPC") {
		t.Fatalf("runServeLoop() error = %v, want serve context", err)
	}
	if !errors.Is(err, serveErr) {
		t.Fatalf("runServeLoop() error = %v, want wrapped serve error", err)
	}
}

func TestNormalizeServeErrorTreatsServerStoppedAsClean(t *testing.T) {
	if err := normalizeServeError(grpc.ErrServerStopped); err != nil {
		t.Fatalf("normalizeServeError(grpc.ErrServerStopped) = %v, want nil", err)
	}
}
