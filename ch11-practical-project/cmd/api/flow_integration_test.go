//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	bookingcontract "example.com/trainingbooking/internal/booking/contract"
	bookingpg "example.com/trainingbooking/internal/booking/infrastructure/postgres"
	catalogcontract "example.com/trainingbooking/internal/catalog/contract"
	"example.com/trainingbooking/internal/catalog/domain"
	catalogpg "example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	identity "example.com/trainingbooking/internal/identity/domain"
	identitypg "example.com/trainingbooking/internal/identity/infrastructure/postgres"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	jst = time.FixedZone("JST", 9*60*60)
	// 現在は2030年5月1日で、レッスンは6月1日10時とする。
	testNow     = time.Date(2030, time.May, 1, 0, 0, 0, 0, jst)
	lessonStart = time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// flow は発火から購読までを実 DB で流すための部品である。
type flow struct {
	t       *testing.T
	pool    *pgxpool.Pool
	app     *app
	lessons *catalogpg.LessonRepository
	tx      *pgtx.Transactor
}

func newFlow(t *testing.T, store outbox.RelayStore) *flow {
	t.Helper()
	pool := dbtest.New(t, bookingpg.Migrations,
		catalogpg.Migrations, outbox.Migrations,
		identitypg.Migrations)
	var logs bytes.Buffer
	return &flow{
		t:    t,
		pool: pool,
		app: newApp(appConfig{
			pool:       pool,
			logger:     slog.New(slog.NewJSONHandler(&logs, nil)),
			clock:      fixedClock{testNow},
			relayStore: store,
		}),
		lessons: catalogpg.NewLessonRepository(pool),
		tx:      pgtx.NewTransactor(pool),
	}
}

func (f *flow) addLesson(capacity int) ids.LessonID {
	f.t.Helper()
	l, err := domain.NewLesson(domain.LessonSpec{
		ID:       ids.NewID[ids.LessonMarker](),
		Title:    "朝のヨガ",
		Start:    lessonStart,
		Duration: time.Hour,
		FeeYen:   3000,
		Capacity: capacity,
		Options: []domain.LessonOption{
			{Name: "シューズ", UnitPriceYen: 300},
		},
	})
	if err != nil {
		f.t.Fatalf("NewLesson: %v", err)
	}
	err = f.tx.Do(f.t.Context(), func(ctx context.Context) error {
		return f.lessons.Add(ctx, l)
	})
	if err != nil {
		f.t.Fatalf("Add: %v", err)
	}
	return l.ID()
}

func (f *flow) request(
	method, path, body string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(
		method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	f.app.router.ServeHTTP(rec, req)
	return rec
}

// place は予約して ID を返す。
func (f *flow) place(
	member ids.MemberID, lesson ids.LessonID,
) string {
	f.t.Helper()
	rec := f.request("POST", "/bookings", fmt.Sprintf(
		`{"member_id":%q,"lesson_id":%q,`+
			`"options":[{"name":"シューズ","quantity":1}]}`,
		member, lesson))
	if rec.Code != http.StatusCreated {
		f.t.Fatalf("予約: %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		f.t.Fatalf("本文: %v", err)
	}
	if body.Status != "pending" {
		f.t.Fatalf("予約の直後の状態 = %s", body.Status)
	}
	return body.ID
}

type listed struct {
	ID       string `json:"id"`
	Status   string `json:"status"`
	TotalYen int64  `json:"total_yen"`
}

// status は GET の一覧から予約を読む。
func (f *flow) status(member ids.MemberID, id string) listed {
	f.t.Helper()
	rec := f.request("GET",
		"/members/"+member.String()+"/bookings", "")
	if rec.Code != http.StatusOK {
		f.t.Fatalf("一覧: %d: %s", rec.Code, rec.Body)
	}
	var body struct {
		Bookings []listed `json:"bookings"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		f.t.Fatalf("本文: %v", err)
	}
	for _, b := range body.Bookings {
		if b.ID == id {
			return b
		}
	}
	f.t.Fatalf("予約が無い: %s", rec.Body)
	return listed{}
}

func (f *flow) drain() {
	f.t.Helper()
	for range 100 {
		n, err := f.app.relay.RunOnce(f.t.Context())
		if err != nil {
			f.t.Fatalf("RunOnce: %v", err)
		}
		if n == 0 {
			f.wantNoUndelivered()
			return
		}
	}
	f.t.Fatal("配信が終わらない")
}

// wantNoUndelivered は未配信が残らないことを確かめる。RunOnce の 0 件
// は受け手の失敗でも返るので、これを見ないと失敗を見逃す。
func (f *flow) wantNoUndelivered() {
	f.t.Helper()
	var n int
	var lastError *string
	err := f.pool.QueryRow(f.t.Context(),
		`SELECT count(*), max(last_error) FROM outbox
		 WHERE published_at IS NULL`).Scan(&n, &lastError)
	if err != nil {
		f.t.Fatalf("QueryRow: %v", err)
	}
	if n != 0 {
		f.t.Fatalf("未配信 = %d 件, 原因 = %v", n, *lastError)
	}
}

func (f *flow) lesson(id ids.LessonID) *domain.Lesson {
	f.t.Helper()
	l, err := f.lessons.FindByID(f.t.Context(), id)
	if err != nil {
		f.t.Fatalf("FindByID: %v", err)
	}
	return l
}

func (f *flow) countEvents(eventType string) int {
	f.t.Helper()
	var n int
	err := f.pool.QueryRow(f.t.Context(),
		`SELECT count(*) FROM outbox WHERE event_type = $1`,
		eventType).Scan(&n)
	if err != nil {
		f.t.Fatalf("QueryRow: %v", err)
	}
	return n
}

// newMember は会員を登録して ID を返す。
func (f *flow) newMember() ids.MemberID {
	f.t.Helper()
	id := ids.NewID[ids.MemberMarker]()
	email, err := identity.NewEmail(id.String() + "@example.com")
	if err != nil {
		f.t.Fatalf("NewEmail: %v", err)
	}
	m, err := identity.NewMember(id, email, "花子")
	if err != nil {
		f.t.Fatalf("NewMember: %v", err)
	}
	err = f.tx.Do(f.t.Context(), func(ctx context.Context) error {
		return identitypg.NewMemberRepository(f.pool).Add(ctx, m)
	})
	if err != nil {
		f.t.Fatalf("Add: %v", err)
	}
	return id
}

func TestBookingFlow_ConfirmedWhenSeatAvailable(t *testing.T) {
	f := newFlow(t, nil)
	lesson := f.addLesson(1)
	member := f.newMember()
	id := f.place(member, lesson)

	// 配信の前は pending のまま見える
	if got := f.status(member, id); got.Status != "pending" {
		t.Fatalf("配信の前 = %s", got.Status)
	}
	f.drain()
	got := f.status(member, id)
	if got.Status != "confirmed" {
		t.Errorf("配信の後 = %s", got.Status)
	}
	// 料金とオプションの単価はレッスンの値である
	if got.TotalYen != 3300 {
		t.Errorf("TotalYen = %d", got.TotalYen)
	}
	if n := f.lesson(lesson).ReservedSeats(); n != 1 {
		t.Errorf("席 = %d", n)
	}
}

func TestBookingFlow_RejectedWhenFull(t *testing.T) {
	f := newFlow(t, nil)
	lesson := f.addLesson(1)
	first, second := f.newMember(), f.newMember()
	firstID := f.place(first, lesson)
	secondID := f.place(second, lesson)
	f.drain()

	if got := f.status(first, firstID); got.Status != "confirmed" {
		t.Errorf("1件目 = %s", got.Status)
	}
	if got := f.status(second, secondID); got.Status != "rejected" {
		t.Errorf("2件目 = %s", got.Status)
	}
	if n := f.lesson(lesson).ReservedSeats(); n != 1 {
		t.Errorf("席 = %d", n)
	}
	// 拒否された予約は時間帯を占めない
	other := f.addLesson(1)
	retry := f.place(second, other)
	f.drain()
	if got := f.status(second, retry); got.Status != "confirmed" {
		t.Errorf("別のレッスン = %s", got.Status)
	}
}

func TestCancel_ReleasesSeat(t *testing.T) {
	f := newFlow(t, nil)
	lesson := f.addLesson(1)
	first, second := f.newMember(), f.newMember()
	id := f.place(first, lesson)
	f.drain()

	rec := f.request("POST", "/bookings/"+id+"/cancel", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("キャンセル: %d: %s", rec.Code, rec.Body)
	}
	// 席が戻るのは配信の後である
	if n := f.lesson(lesson).ReservedSeats(); n != 1 {
		t.Errorf("配信の前の席 = %d", n)
	}
	f.drain()
	if n := f.lesson(lesson).ReservedSeats(); n != 0 {
		t.Errorf("配信の後の席 = %d", n)
	}
	if got := f.status(first, id); got.Status != "cancelled" {
		t.Errorf("状態 = %s", got.Status)
	}

	// 戻った席を別の会員が確保できる
	next := f.place(second, lesson)
	f.drain()
	if got := f.status(second, next); got.Status != "confirmed" {
		t.Errorf("戻った席の確保 = %s", got.Status)
	}
}

// pending のキャンセルは席の確保と行き違うので拒否する。
func TestCancel_PendingIsConflict(t *testing.T) {
	f := newFlow(t, nil)
	lesson := f.addLesson(1)
	member := f.newMember()
	id := f.place(member, lesson)
	rec := f.request("POST", "/bookings/"+id+"/cancel", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d: %s", rec.Code, rec.Body)
	}
}

func TestPlaceBooking_UnknownLessonIsBadRequest(t *testing.T) {
	f := newFlow(t, nil)
	rec := f.request("POST", "/bookings", fmt.Sprintf(
		`{"member_id":%q,"lesson_id":%q}`,
		f.newMember(), ids.NewID[ids.LessonMarker]()))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d: %s", rec.Code, rec.Body)
	}
}

func TestPlaceBooking_UnknownMemberIsBadRequest(t *testing.T) {
	f := newFlow(t, nil)
	lesson := f.addLesson(1)
	rec := f.request("POST", "/bookings", fmt.Sprintf(
		`{"member_id":%q,"lesson_id":%q}`,
		ids.NewID[ids.MemberMarker](), lesson))
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d: %s", rec.Code, rec.Body)
	}
	n := f.countEvents(bookingcontract.EventBookingPlaced)
	if n != 0 {
		t.Errorf("BookingPlaced = %d 件", n)
	}
}

func TestPlaceBooking_OverlapIsConflict(t *testing.T) {
	f := newFlow(t, nil)
	first, second := f.addLesson(5), f.addLesson(5)
	member := f.newMember()
	f.place(member, first)
	rec := f.request("POST", "/bookings", fmt.Sprintf(
		`{"member_id":%q,"lesson_id":%q}`, member, second))
	if rec.Code != http.StatusConflict {
		t.Errorf("status = %d: %s", rec.Code, rec.Body)
	}
}

// markFails は指定の種類のイベントで、最初の MarkPublished だけを
// 失敗させる。受け手の処理の後、配信済みの記録の前に止まった状況を
// 作る。
type markFails struct {
	outbox.RelayStore
	eventType string
	byID      map[int64]string
	failed    int
}

func (s *markFails) Deliverable(
	ctx context.Context, limit int,
) ([]outbox.Delivery, error) {
	got, err := s.RelayStore.Deliverable(ctx, limit)
	for _, d := range got {
		s.byID[d.ID] = d.EventType
	}
	return got, err
}

func (s *markFails) MarkPublished(
	ctx context.Context, id int64,
) error {
	if s.byID[id] == s.eventType && s.failed == 0 {
		s.failed++
		return errors.New("stopped before marking")
	}
	return s.RelayStore.MarkPublished(ctx, id)
}

// runUntilEmpty は記録の失敗を許して配信を終わりまで進める。
func (f *flow) runUntilEmpty() {
	f.t.Helper()
	for range 100 {
		n, err := f.app.relay.RunOnce(f.t.Context())
		if n == 0 && err == nil {
			f.wantNoUndelivered()
			return
		}
	}
	f.t.Fatal("配信が終わらない")
}

func newRedeliveryFlow(
	t *testing.T, eventType string,
) (*flow, *markFails) {
	t.Helper()
	store := &markFails{
		eventType: eventType,
		byID:      make(map[int64]string),
	}
	f := newFlow(t, store)
	store.RelayStore = outbox.NewStore(f.pool)
	return f, store
}

func TestRelay_RedeliveryIsIdempotent(t *testing.T) {
	f, store := newRedeliveryFlow(
		t, bookingcontract.EventBookingPlaced)
	lesson := f.addLesson(2)
	member := f.newMember()
	id := f.place(member, lesson)
	f.runUntilEmpty()

	if store.failed != 1 {
		t.Fatalf("記録の失敗 = %d 回", store.failed)
	}
	if n := f.lesson(lesson).ReservedSeats(); n != 1 {
		t.Errorf("席 = %d", n)
	}
	// 席を動かした保存も1回だけである
	if v := f.lesson(lesson).Version(); v != 1 {
		t.Errorf("Lesson の版 = %d", v)
	}
	n := f.countEvents(catalogcontract.EventSeatReserved)
	if n != 1 {
		t.Errorf("SeatReserved = %d 件", n)
	}
	if got := f.status(member, id); got.Status != "confirmed" {
		t.Errorf("状態 = %s", got.Status)
	}
}

func TestRelay_RedeliveredSeatResultConfirmsOnce(t *testing.T) {
	f, store := newRedeliveryFlow(
		t, catalogcontract.EventSeatReserved)
	lesson := f.addLesson(1)
	member := f.newMember()
	id := f.place(member, lesson)
	f.runUntilEmpty()

	if store.failed != 1 {
		t.Fatalf("記録の失敗 = %d 回", store.failed)
	}
	if got := f.status(member, id); got.Status != "confirmed" {
		t.Errorf("状態 = %s", got.Status)
	}
	var version int
	err := f.pool.QueryRow(t.Context(),
		`SELECT version FROM bookings WHERE id = $1`, id).
		Scan(&version)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if version != 1 {
		t.Errorf("Booking の版 = %d", version)
	}
}

func TestRelay_RedeliveredCancelReleasesOnce(t *testing.T) {
	f, store := newRedeliveryFlow(
		t, bookingcontract.EventBookingCancelled)
	lesson := f.addLesson(2)
	member, other := f.newMember(), f.newMember()
	id := f.place(member, lesson)
	f.place(other, lesson)
	f.runUntilEmpty()

	rec := f.request("POST", "/bookings/"+id+"/cancel", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("キャンセル: %d: %s", rec.Code, rec.Body)
	}
	f.runUntilEmpty()
	if store.failed != 1 {
		t.Fatalf("記録の失敗 = %d 回", store.failed)
	}
	if n := f.lesson(lesson).ReservedSeats(); n != 1 {
		t.Errorf("席 = %d", n)
	}
	// 予約2件で2版、戻すので1版。再配信では増えない
	if v := f.lesson(lesson).Version(); v != 3 {
		t.Errorf("Lesson の版 = %d", v)
	}
}

// HTTP のサーバーと動いている配信処理を、実 DB につないで通す。
func TestEndToEnd_PlaceConfirmCancel(t *testing.T) {
	f := newFlow(t, nil)
	lesson := f.addLesson(1)
	member, other := f.newMember(), f.newMember()
	server := httptest.NewServer(f.app.router)
	defer server.Close()
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan error, 1)
	go func() { done <- f.app.relay.Run(ctx) }()
	defer func() {
		cancel()
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Errorf("Run: %v", err)
		}
	}()

	post := func(path, body string) (int, map[string]any) {
		t.Helper()
		res, err := http.Post(server.URL+path,
			"application/json", strings.NewReader(body))
		if err != nil {
			t.Fatalf("POST %s: %v", path, err)
		}
		defer func() { _ = res.Body.Close() }()
		var got map[string]any
		err = json.NewDecoder(res.Body).Decode(&got)
		if err != nil {
			t.Fatalf("本文: %v", err)
		}
		return res.StatusCode, got
	}
	body := func(m ids.MemberID) string {
		return fmt.Sprintf(`{"member_id":%q,"lesson_id":%q}`,
			m, lesson)
	}

	status, got := post("/bookings", body(member))
	id, _ := got["id"].(string)
	if status != http.StatusCreated || got["status"] != "pending" {
		t.Fatalf("予約: %d %v", status, got)
	}
	f.waitStatus(member, id, "confirmed")

	status, got = post("/bookings", body(other))
	rejected, _ := got["id"].(string)
	if status != http.StatusCreated {
		t.Fatalf("満席の予約: %d %v", status, got)
	}
	f.waitStatus(other, rejected, "rejected")

	status, got = post("/bookings/"+id+"/cancel", "")
	if status != http.StatusOK || got["status"] != "cancelled" {
		t.Fatalf("キャンセル: %d %v", status, got)
	}
	deadline := time.Now().Add(30 * time.Second)
	for f.lesson(lesson).ReservedSeats() != 0 {
		if time.Now().After(deadline) {
			t.Fatal("席が戻らない")
		}
		time.Sleep(10 * time.Millisecond)
	}

	// 戻った席は別の予約で確保できる
	status, got = post("/bookings", body(other))
	next, _ := got["id"].(string)
	if status != http.StatusCreated {
		t.Fatalf("戻った席の予約: %d %v", status, got)
	}
	f.waitStatus(other, next, "confirmed")
}

func (f *flow) waitStatus(
	member ids.MemberID, id, want string,
) {
	f.t.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for {
		got := f.status(member, id)
		if got.Status == want {
			return
		}
		if time.Now().After(deadline) {
			f.t.Fatalf("状態 = %s, want %s", got.Status, want)
		}
		time.Sleep(10 * time.Millisecond)
	}
}
