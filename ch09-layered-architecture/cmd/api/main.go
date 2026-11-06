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

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/fixed"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/booking/presentation"
	"example.com/trainingbooking/internal/platform/logging"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultDatabaseURL = "postgres://dev:dev@localhost:5432/" +
		"trainingbooking?sslmode=disable"
	defaultAddr     = "localhost:8080"
	shutdownTimeout = 10 * time.Second
	requestTimeout  = 5 * time.Second
	lessonFeeYen    = 3000
	shoesPriceYen   = 300
	towelPriceYen   = 100
)

// #region system-clock

// systemClock は実際の現在時刻を返す。
type systemClock struct{}

func (systemClock) Now() time.Time { return time.Now() }

// #endregion

func main() {
	logger := slog.New(logging.NewContextHandler(
		slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(logger); err != nil {
		logger.Error("api stopped", "error", err)
		os.Exit(1)
	}
}

// #region run

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

	router, err := newRouter(pool, logger)
	if err != nil {
		return err
	}
	server := &http.Server{
		Addr:              cmp.Or(os.Getenv("ADDR"), defaultAddr),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return serve(ctx, server, logger)
}

// #endregion

// #region new-router

func newRouter(
	pool *pgxpool.Pool, logger *slog.Logger,
) (http.Handler, error) {
	clock := systemClock{}
	terms, err := lessonTerms(clock.Now())
	if err != nil {
		return nil, err
	}
	repo := postgres.NewBookingRepository(pool)
	queries := postgres.NewBookingQueries(pool)
	tx := pgtx.NewTransactor(pool)
	outbox := postgres.NewLoggingOutbox(logger)
	return presentation.NewRouter(presentation.Deps{
		Place: application.NewPlaceBooking(
			repo, queries, outbox, tx, clock,
			fixed.NewCoupons(), terms),
		Cancel: application.NewCancelBooking(
			repo, outbox, tx, clock),
		Queries:  queries,
		Logger:   logger,
		Location: terms.Location,
		Timeout:  requestTimeout,
	}), nil
}

// #endregion

// lessonTerms はレッスンの条件を固定で返す。時間帯は起動した日の
// 30日後の10時に置き、いつ動かしても未来の日時にする。
func lessonTerms(now time.Time) (application.LessonTerms, error) {
	fee, err := domain.NewMoney(lessonFeeYen)
	if err != nil {
		return application.LessonTerms{}, err
	}
	shoes, err := domain.NewMoney(shoesPriceYen)
	if err != nil {
		return application.LessonTerms{}, err
	}
	towel, err := domain.NewMoney(towelPriceYen)
	if err != nil {
		return application.LessonTerms{}, err
	}
	jst := time.FixedZone("JST", 9*60*60)
	day := now.In(jst).AddDate(0, 0, 30)
	return application.LessonTerms{
		Start: time.Date(
			day.Year(), day.Month(), day.Day(), 10, 0, 0, 0, jst),
		Duration: time.Hour,
		Fee:      fee,
		Location: jst,
		Options: []application.OfferedOption{
			{Name: "シューズ", UnitPrice: shoes},
			{Name: "タオル", UnitPrice: towel},
		},
	}, nil
}

// #region serve

// serve は ctx が終わるまで受け付け、処理中のリクエストを待って
// 止まる。
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

// #endregion

// #region shutdown

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

// #endregion
