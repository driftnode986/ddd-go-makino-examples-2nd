// Command api は予約システムの HTTP サーバーである。
package main

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"example.com/trainingbooking/internal/platform/logging"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultDatabaseURL = "postgres://dev:dev@localhost:5432/" +
		"trainingbooking?sslmode=disable"
	defaultAddr     = "localhost:8080"
	shutdownTimeout = 10 * time.Second
)

// systemClock は実際の現在時刻を返す。
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

func main() {
	logger := slog.New(logging.NewContextHandler(
		slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	ctx, stop := signal.NotifyContext(context.Background(),
		os.Interrupt, syscall.SIGTERM)
	defer stop()

	url := cmp.Or(os.Getenv("DATABASE_URL"), defaultDatabaseURL)
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		return fmt.Errorf("connect: %w", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		return fmt.Errorf("ping: %w", err)
	}

	a := newApp(appConfig{
		pool:   pool,
		logger: logger,
		clock:  systemClock{},
	})
	relayDone := make(chan error, 1)
	go func() { relayDone <- a.relay.Run(ctx) }()

	server := &http.Server{
		Addr:              cmp.Or(os.Getenv("ADDR"), defaultAddr),
		Handler:           a.router,
		ReadHeaderTimeout: 5 * time.Second,
	}
	err = serve(ctx, server, logger)
	// サーバーが先に止まっても配信処理を止め、終わるまで待つ。
	// 接続を閉じるのはその後である
	stop()
	if relayErr := <-relayDone; !errors.Is(
		relayErr, context.Canceled) {
		err = errors.Join(err, relayErr)
	}
	return err
}

// serve は ctx が終わるまで受け付け、処理中のリクエストを待って止まる。
func serve(
	ctx context.Context, server *http.Server, logger *slog.Logger,
) error {
	errs := make(chan error, 1)
	go func() {
		logger.Info("listening", "addr", server.Addr)
		errs <- server.ListenAndServe()
	}()
	select {
	case err := <-errs:
		return err
	case <-ctx.Done():
	}
	logger.Info("shutting down")
	if err := shutdown(server, shutdownTimeout); err != nil {
		return err
	}
	if err := <-errs; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

// shutdown は処理中のリクエストを timeout まで待つ。残ったものは
// 接続を閉じて中断する。DB を待ったまま残ると接続が返らず、
// プールを閉じる処理が止まる。
func shutdown(server *http.Server, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(
		context.Background(), timeout)
	defer cancel()
	err := server.Shutdown(ctx)
	if err != nil {
		return errors.Join(
			fmt.Errorf("shutdown: %w", err), server.Close())
	}
	return nil
}
