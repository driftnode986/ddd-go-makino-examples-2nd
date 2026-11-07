package main

import (
	"log/slog"
	"net/http"
	"time"

	bookingapp "example.com/trainingbooking/internal/booking/application"
	bookingcontract "example.com/trainingbooking/internal/booking/contract"
	"example.com/trainingbooking/internal/booking/infrastructure/acl"
	"example.com/trainingbooking/internal/booking/infrastructure/fixed"
	bookingpg "example.com/trainingbooking/internal/booking/infrastructure/postgres"
	"example.com/trainingbooking/internal/booking/presentation"
	catalogapp "example.com/trainingbooking/internal/catalog/application"
	catalogcontract "example.com/trainingbooking/internal/catalog/contract"
	catalogpg "example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	identityapp "example.com/trainingbooking/internal/identity/application"
	identitypg "example.com/trainingbooking/internal/identity/infrastructure/postgres"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
)

// clock は Booking と Catalog の Clock を兼ねる。
type clock interface {
	Now() time.Time
}

// appConfig は組み立てに使う部品である。
type appConfig struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
	clock  clock
	// relayStore は配信処理の読み書きを差し替える。テストで途中の
	// 失敗を作るためのもので、nil なら outbox の Store を使う。
	relayStore outbox.RelayStore
}

// app は組み立てたサーバーと配信処理である。
type app struct {
	router http.Handler
	relay  *outbox.Relay
}

// newApp はコンテキストを組み立てる。別のコンテキストの contract を
// 実装につなぐ場所はここだけである。
func newApp(cfg appConfig) *app {
	tx := pgtx.NewTransactor(cfg.pool)
	store := outbox.NewStore(cfg.pool)
	processed := outbox.NewProcessedEvents(cfg.pool)

	lessons := catalogpg.NewLessonRepository(cfg.pool)
	seats := catalogapp.NewBookingEventHandler(
		catalogapp.NewReserveSeat(lessons,
			catalogpg.NewOutboxWriter(store),
			processed, tx, cfg.clock),
		catalogapp.NewReleaseSeat(lessons, processed, tx))

	bookings := bookingpg.NewBookingRepository(cfg.pool)
	queries := bookingpg.NewBookingQueries(cfg.pool)
	writer := bookingpg.NewOutboxWriter(store)
	results := bookingapp.NewSeatResultHandler(
		bookings, processed, tx)
	jst := time.FixedZone("JST", 9*60*60)
	catalog := acl.NewLessonCatalog(
		catalogapp.NewLessonReader(lessons), jst)
	members := acl.NewMemberDirectory(identityapp.NewMemberReader(
		identitypg.NewMemberRepository(cfg.pool)))

	var relayStore outbox.RelayStore = store
	if cfg.relayStore != nil {
		relayStore = cfg.relayStore
	}
	relay := outbox.NewRelay(
		relayStore, cfg.logger, outbox.RelayOptions{})
	subscribe(relay, seats, results)

	router := presentation.NewRouter(presentation.Deps{
		Place: bookingapp.NewPlaceBooking(
			bookings, queries, writer, tx, cfg.clock,
			fixed.NewCoupons(), catalog, members),
		Cancel: bookingapp.NewCancelBooking(
			bookings, writer, tx, cfg.clock),
		Queries:  queries,
		Logger:   cfg.logger,
		Location: jst,
		Timeout:  requestTimeout,
	})
	return &app{router: router, relay: relay}
}

// #region subscribe

// subscribe はイベントの種類ごとに購読者を登録する。
func subscribe(
	relay *outbox.Relay,
	seats *catalogapp.BookingEventHandler,
	results *bookingapp.SeatResultHandler,
) {
	relay.Subscribe(bookingcontract.EventBookingPlaced,
		catalogapp.Subscriber, outbox.On(seats.OnBookingPlaced))
	relay.Subscribe(bookingcontract.EventBookingCancelled,
		catalogapp.Subscriber, outbox.On(seats.OnBookingCancelled))
	relay.Subscribe(catalogcontract.EventSeatReserved,
		bookingapp.Subscriber, outbox.On(results.OnSeatReserved))
	relay.Subscribe(catalogcontract.EventSeatRejected,
		bookingapp.Subscriber, outbox.On(results.OnSeatRejected))
}

// #endregion
