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

	bookingapp "example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/infrastructure/acl"
	bookingpg "example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/booking/presentation"
	catalogapp "example.com/trainingbooking/internal/catalog/application"
	catalogpg "example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	"example.com/trainingbooking/internal/platform/logging"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
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

	server := &http.Server{
		Addr:              cmp.Or(os.Getenv("ADDR"), defaultAddr),
		Handler:           newRouter(pool, logger),
		ReadHeaderTimeout: 5 * time.Second,
	}
	return serve(ctx, server, logger)
}

func newRouter(
	pool *pgxpool.Pool, logger *slog.Logger,
) http.Handler {
	repo := bookingpg.NewBookingRepository(pool)
	tx := pgtx.NewTransactor(pool)
	outbox := bookingpg.NewLoggingOutbox(logger)
	clock := systemClock{}
	lessons := acl.NewLessonCatalog(
		catalogapp.NewLessonReader(
			catalogpg.NewLessonRepository(pool)),
		time.FixedZone("JST", 9*60*60))
	return presentation.NewRouter(presentation.Deps{
		Place: bookingapp.NewPlaceBooking(
			repo, outbox, tx, clock, lessons),
		Cancel: bookingapp.NewCancelBooking(
			repo, outbox, tx, clock),
		Queries: bookingpg.NewBookingQueries(pool),
		Logger:  logger,
	})
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
	shutdown, cancel := context.WithTimeout(
		context.Background(), shutdownTimeout)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	if err := <-errs; !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}
