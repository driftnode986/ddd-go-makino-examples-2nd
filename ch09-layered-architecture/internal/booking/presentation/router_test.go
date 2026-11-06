package presentation_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/fixed"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
	"example.com/trainingbooking/internal/booking/presentation"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/logging"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

var (
	jst = time.FixedZone("JST", 9*60*60)
	now = time.Date(2030, time.May, 1, 0, 0, 0, 0, jst)
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// stubUseCases は決めた結果を返すユースケースである。
type stubUseCases struct {
	place func(
		application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error)
	cancel func(
		application.CancelBookingInput,
	) (application.CancelBookingOutput, error)
	list func(ids.MemberID) ([]application.BookingView, error)
}

func (s stubUseCases) Execute(
	_ context.Context, in application.PlaceBookingInput,
) (application.PlaceBookingOutput, error) {
	return s.place(in)
}

func (s stubUseCases) ListByMember(
	_ context.Context, member ids.MemberID,
) ([]application.BookingView, error) {
	return s.list(member)
}

type stubCancel struct{ stubUseCases }

func (s stubCancel) Execute(
	_ context.Context, in application.CancelBookingInput,
) (application.CancelBookingOutput, error) {
	return s.cancel(in)
}

func newRouter(
	s stubUseCases, buf *bytes.Buffer,
) http.Handler {
	logger := slog.New(logging.NewContextHandler(
		slog.NewJSONHandler(buf, nil)))
	return presentation.NewRouter(presentation.Deps{
		Place:    s,
		Cancel:   stubCancel{s},
		Queries:  s,
		Logger:   logger,
		Location: jst,
		Timeout:  time.Second,
	})
}

// postValid は正しい本文で POST /bookings を呼ぶ。
func postValid(
	s stubUseCases, buf *bytes.Buffer,
) *httptest.ResponseRecorder {
	return do(newRouter(s, buf), "POST", "/bookings", validBody)
}

func do(
	h http.Handler, method, path, body string,
) *httptest.ResponseRecorder {
	req := httptest.NewRequest(
		method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decode(
	t *testing.T, rec *httptest.ResponseRecorder,
) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &m); err != nil {
		t.Fatalf("本文を読めない: %v: %s", err, rec.Body)
	}
	return m
}

// logRecords はログを1行ずつ読む。
func logRecords(t *testing.T, buf *bytes.Buffer) []map[string]any {
	t.Helper()
	var out []map[string]any
	for line := range strings.Lines(buf.String()) {
		var m map[string]any
		if err := json.Unmarshal([]byte(line), &m); err != nil {
			t.Fatalf("ログを読めない: %v: %s", err, line)
		}
		out = append(out, m)
	}
	return out
}

const validBody = `{
	"member_id": "0196f5a0-0000-7000-8000-000000000001",
	"lesson_id": "0196f5a0-0000-7000-8000-000000000002",
	"options": [{"name": "シューズ", "quantity": 2}]
}`

func TestPostBookings_Created(t *testing.T) {
	var got application.PlaceBookingInput
	s := stubUseCases{place: func(
		in application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error) {
		got = in
		return application.PlaceBookingOutput{
			BookingID: "b1", Status: "pending",
		}, nil
	}}
	var buf bytes.Buffer
	rec := postValid(s, &buf)

	if rec.Code != http.StatusCreated {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if ct := rec.Header().Get("Content-Type"); !strings.HasPrefix(
		ct, "application/json") {
		t.Errorf("Content-Type = %s", ct)
	}
	body := decode(t, rec)
	if body["id"] != "b1" || body["status"] != "pending" {
		t.Errorf("body = %v", body)
	}
	if got.MemberID != "0196f5a0-0000-7000-8000-000000000001" ||
		got.LessonID != "0196f5a0-0000-7000-8000-000000000002" ||
		len(got.Options) != 1 || got.Options[0].Name != "シューズ" ||
		got.Options[0].Quantity != 2 {
		t.Errorf("input = %+v", got)
	}
}

func TestPostBookings_PassesCouponCode(t *testing.T) {
	cases := []struct{ name, body, want string }{
		{"省略", validBody, ""},
		{"指定", strings.Replace(validBody, "{",
			`{"coupon_code":"WELCOME10",`, 1), "WELCOME10"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got application.PlaceBookingInput
			s := stubUseCases{place: func(
				in application.PlaceBookingInput,
			) (application.PlaceBookingOutput, error) {
				got = in
				return application.PlaceBookingOutput{}, nil
			}}
			var buf bytes.Buffer
			router := newRouter(s, &buf)
			rec := do(router, "POST", "/bookings", c.body)
			if rec.Code != http.StatusCreated {
				t.Fatalf("status = %d: %s", rec.Code, rec.Body)
			}
			if got.CouponCode != c.want {
				t.Errorf("CouponCode = %q", got.CouponCode)
			}
		})
	}
}

func TestPostBookings_BadJSON(t *testing.T) {
	cases := []struct{ name, body string }{
		{"空", ""},
		{"構文の誤り", `{"member_id":`},
		{"型の誤り", `{"options":"シューズ"}`},
		{"知らない項目", `{"fee_yen":1}`},
		{"割引率は受け取らない", `{"discount_percent":10}`},
		{"開始は受け取らない", `{"start":"2030-06-01T10:00:00+09:00"}`},
		{"長さは受け取らない", `{"duration_minutes":60}`},
		{"単価は受け取らない",
			`{"options":[{"name":"x","unit_price_yen":1}]}`},
		{"2つの値", `{} {}`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			called := false
			s := stubUseCases{place: func(
				application.PlaceBookingInput,
			) (application.PlaceBookingOutput, error) {
				called = true
				return application.PlaceBookingOutput{}, nil
			}}
			var buf bytes.Buffer
			router := newRouter(s, &buf)
			rec := do(router, "POST", "/bookings", c.body)
			if rec.Code != http.StatusBadRequest {
				t.Errorf("status = %d: %s", rec.Code, rec.Body)
			}
			if called {
				t.Error("ユースケースが呼ばれた")
			}
			if decode(t, rec)["kind"] != "invalid" {
				t.Errorf("body = %s", rec.Body)
			}
		})
	}
}

func TestPostBookings_TooLargeBody(t *testing.T) {
	s := stubUseCases{}
	var buf bytes.Buffer
	body := `{"member_id":"` + strings.Repeat("a", 1<<21) + `"}`
	rec := do(newRouter(s, &buf), "POST", "/bookings", body)
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
}

func TestStatus_ByKind(t *testing.T) {
	cases := []struct {
		kind application.Kind
		want int
	}{
		{application.KindInvalid, http.StatusBadRequest},
		{application.KindNotFound, http.StatusNotFound},
		{application.KindConflict, http.StatusConflict},
		{application.KindInternal, http.StatusInternalServerError},
	}
	for _, c := range cases {
		t.Run(string(c.kind), func(t *testing.T) {
			s := stubUseCases{place: func(
				application.PlaceBookingInput,
			) (application.PlaceBookingOutput, error) {
				return application.PlaceBookingOutput{},
					&application.ApplicationError{
						Kind:    c.kind,
						Message: "外に出す文",
						Cause:   errors.New("原因"),
					}
			}}
			var buf bytes.Buffer
			rec := postValid(s, &buf)
			if rec.Code != c.want {
				t.Errorf("status = %d", rec.Code)
			}
			body := decode(t, rec)
			if body["kind"] != string(c.kind) ||
				body["message"] != "外に出す文" {
				t.Errorf("body = %v", body)
			}
			if strings.Contains(rec.Body.String(), "原因") {
				t.Errorf("本文に原因が出た: %s", rec.Body)
			}
		})
	}
}

func TestInternalError_HidesCauseButLogsIt(t *testing.T) {
	const cause = "dial tcp secret-host:5432: refused"
	s := stubUseCases{place: func(
		application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error) {
		return application.PlaceBookingOutput{},
			&application.ApplicationError{
				Kind:    application.KindInternal,
				Message: "内部エラーが発生しました",
				Cause:   errors.New(cause),
			}
	}}
	var buf bytes.Buffer
	rec := postValid(s, &buf)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret-host") {
		t.Errorf("本文に原因が出た: %s", rec.Body)
	}
	if !strings.Contains(buf.String(), cause) {
		t.Errorf("ログに原因が無い: %s", buf.String())
	}
	levels := levelsOf(t, &buf, "request failed")
	if len(levels) != 1 || levels[0] != "ERROR" {
		t.Errorf("levels = %v", levels)
	}
}

// ApplicationError でないエラーも 500 にして原因を隠す。
func TestUnknownError_IsInternal(t *testing.T) {
	s := stubUseCases{list: func(
		ids.MemberID,
	) ([]application.BookingView, error) {
		return nil, errors.New("secret-host")
	}}
	var buf bytes.Buffer
	path := "/members/" + ids.NewID[ids.MemberMarker]().String() +
		"/bookings"
	rec := do(newRouter(s, &buf), "GET", path, "")
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret-host") {
		t.Errorf("本文に原因が出た: %s", rec.Body)
	}
	if !strings.Contains(buf.String(), "secret-host") {
		t.Errorf("ログに原因が無い: %s", buf.String())
	}
}

// 利用者の切断と期限切れは 500 のままで、ログの水準を下げる。
func TestContextErrors_AreLoggedAsWarning(t *testing.T) {
	for _, cause := range []error{
		context.Canceled, context.DeadlineExceeded,
	} {
		t.Run(cause.Error(), func(t *testing.T) {
			s := stubUseCases{place: func(
				application.PlaceBookingInput,
			) (application.PlaceBookingOutput, error) {
				return application.PlaceBookingOutput{},
					&application.ApplicationError{
						Kind:    application.KindInternal,
						Message: "内部エラーが発生しました",
						Cause:   cause,
					}
			}}
			var buf bytes.Buffer
			rec := postValid(s, &buf)
			if rec.Code != http.StatusInternalServerError {
				t.Errorf("status = %d", rec.Code)
			}
			levels := levelsOf(t, &buf, "request failed")
			if len(levels) != 1 || levels[0] != "WARN" {
				t.Errorf("levels = %v", levels)
			}
		})
	}
}

func levelsOf(
	t *testing.T, buf *bytes.Buffer, msg string,
) []any {
	t.Helper()
	var levels []any
	for _, r := range logRecords(t, buf) {
		if r["msg"] == msg {
			levels = append(levels, r["level"])
		}
	}
	return levels
}

// 4xx は利用者の誤りなので、原因のログは記録しない。
func TestClientError_IsNotLoggedAsFailure(t *testing.T) {
	s := stubUseCases{place: func(
		application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error) {
		return application.PlaceBookingOutput{},
			&application.ApplicationError{
				Kind:    application.KindConflict,
				Message: "重複",
				Cause:   domain.ErrOverlap,
			}
	}}
	var buf bytes.Buffer
	postValid(s, &buf)
	if l := levelsOf(t, &buf, "request failed"); len(l) != 0 {
		t.Errorf("levels = %v", l)
	}
}

func TestPostCancel(t *testing.T) {
	var got application.CancelBookingInput
	s := stubUseCases{cancel: func(
		in application.CancelBookingInput,
	) (application.CancelBookingOutput, error) {
		got = in
		return application.CancelBookingOutput{
			BookingID: in.BookingID, Status: "cancelled",
		}, nil
	}}
	var buf bytes.Buffer
	rec := do(newRouter(s, &buf), "POST", "/bookings/b1/cancel", "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	if got.BookingID != "b1" {
		t.Errorf("input = %+v", got)
	}
	body := decode(t, rec)
	if body["id"] != "b1" || body["status"] != "cancelled" {
		t.Errorf("body = %v", body)
	}
}

func TestGetBookings(t *testing.T) {
	member := ids.NewID[ids.MemberMarker]()
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	s := stubUseCases{list: func(
		m ids.MemberID,
	) ([]application.BookingView, error) {
		if !m.Equal(member) {
			t.Errorf("member = %s", m)
		}
		return []application.BookingView{{
			ID:       "b1",
			LessonID: "l1",
			Start:    start,
			End:      start.Add(time.Hour),
			Status:   "pending",
			TotalYen: 3000,
			Options: []application.BookingOptionView{
				{Name: "シューズ", Quantity: 2, SubtotalYen: 600},
			},
		}}, nil
	}}
	var buf bytes.Buffer
	path := "/members/" + member.String() + "/bookings"
	rec := do(newRouter(s, &buf), "GET", path, "")
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d: %s", rec.Code, rec.Body)
	}
	bookings, ok := decode(t, rec)["bookings"].([]any)
	if !ok || len(bookings) != 1 {
		t.Fatalf("body = %s", rec.Body)
	}
	got, ok := bookings[0].(map[string]any)
	if !ok {
		t.Fatalf("body = %s", rec.Body)
	}
	if got["id"] != "b1" || got["lesson_id"] != "l1" ||
		got["status"] != "pending" || got["total_yen"] != 3000.0 ||
		got["start"] != "2030-06-01T10:00:00+09:00" ||
		got["end"] != "2030-06-01T11:00:00+09:00" {
		t.Errorf("booking = %v", got)
	}
	const options = `"options":[` +
		`{"name":"シューズ","quantity":2,"subtotal_yen":600}]`
	if !strings.Contains(rec.Body.String(), options) {
		t.Errorf("明細: %s", rec.Body)
	}
}

// 同じ瞬間なら、DB の接続のタイムゾーンによらず同じ文字列を返す。
func TestGetBookings_StartIsInLocation(t *testing.T) {
	start := time.Date(2030, time.June, 1, 1, 0, 0, 0, time.UTC)
	s := stubUseCases{list: func(
		ids.MemberID,
	) ([]application.BookingView, error) {
		return []application.BookingView{{
			Start: start, End: start.Add(time.Hour),
		}}, nil
	}}
	var buf bytes.Buffer
	member := ids.NewID[ids.MemberMarker]().String()
	rec := do(newRouter(s, &buf), "GET",
		"/members/"+member+"/bookings", "")
	const want = `"start":"2030-06-01T10:00:00+09:00"`
	if !strings.Contains(rec.Body.String(), want) {
		t.Errorf("body = %s", rec.Body)
	}
}

// 期限が来ると ctx が終わり、500 と WARN のログになる。
func TestTimeout_EndsTheRequest(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	router := presentation.NewRouter(presentation.Deps{
		Place:    blockingPlace{},
		Logger:   logger,
		Location: jst,
		Timeout:  10 * time.Millisecond,
	})
	rec := do(router, "POST", "/bookings", validBody)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	levels := levelsOf(t, &buf, "request failed")
	if len(levels) != 1 || levels[0] != "WARN" {
		t.Errorf("levels = %v", levels)
	}
}

// blockingPlace は ctx が終わるまで返らない。応答の遅い DB の代わりである。
type blockingPlace struct{}

func (blockingPlace) Execute(
	ctx context.Context, _ application.PlaceBookingInput,
) (application.PlaceBookingOutput, error) {
	<-ctx.Done()
	return application.PlaceBookingOutput{}, ctx.Err()
}

func TestGetBookings_NoOptionsIsArray(t *testing.T) {
	s := stubUseCases{list: func(
		ids.MemberID,
	) ([]application.BookingView, error) {
		return []application.BookingView{{ID: "b1"}}, nil
	}}
	var buf bytes.Buffer
	path := "/members/" + ids.NewID[ids.MemberMarker]().String() +
		"/bookings"
	rec := do(newRouter(s, &buf), "GET", path, "")
	if !strings.Contains(rec.Body.String(), `"options":[]`) {
		t.Errorf("body = %s", rec.Body)
	}
}

func TestGetBookings_EmptyIsArray(t *testing.T) {
	s := stubUseCases{list: func(
		ids.MemberID,
	) ([]application.BookingView, error) {
		return []application.BookingView{}, nil
	}}
	var buf bytes.Buffer
	path := "/members/" + ids.NewID[ids.MemberMarker]().String() +
		"/bookings"
	rec := do(newRouter(s, &buf), "GET", path, "")
	if !strings.Contains(rec.Body.String(), `"bookings":[]`) {
		t.Errorf("body = %s", rec.Body)
	}
}

func TestGetBookings_InvalidMemberID(t *testing.T) {
	s := stubUseCases{}
	var buf bytes.Buffer
	rec := do(newRouter(s, &buf), "GET", "/members/x/bookings", "")
	if rec.Code != http.StatusBadRequest {
		t.Errorf("status = %d", rec.Code)
	}
	if decode(t, rec)["kind"] != "invalid" {
		t.Errorf("body = %s", rec.Body)
	}
}

func TestRouter_MethodNotAllowed(t *testing.T) {
	var buf bytes.Buffer
	router := newRouter(stubUseCases{}, &buf)
	rec := do(router, "GET", "/bookings", "")
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d", rec.Code)
	}
}

// Recovery は RequestID とアクセスログの内側にある。panic の記録にも
// アクセスログにも同じ request_id が付き、利用者には 500 が返る。
func TestMiddleware_PanicReachesRecoveryAndLogs(t *testing.T) {
	s := stubUseCases{place: func(
		application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error) {
		panic("secret-panic-value")
	}}
	var buf bytes.Buffer
	rec := postValid(s, &buf)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", rec.Code)
	}
	if strings.Contains(rec.Body.String(), "secret-panic-value") {
		t.Errorf("本文に panic の値が出た: %s", rec.Body)
	}
	if decode(t, rec)["kind"] != "internal" {
		t.Errorf("body = %s", rec.Body)
	}
	id := rec.Header().Get("X-Request-ID")
	if id == "" {
		t.Fatal("X-Request-ID が無い")
	}
	records := logRecords(t, &buf)
	if len(records) != 2 {
		t.Fatalf("records = %v", records)
	}
	panicked, access := records[0], records[1]
	if panicked["msg"] != "panic" || panicked["request_id"] != id ||
		panicked["panic"] != "secret-panic-value" ||
		panicked["level"] != "ERROR" {
		t.Errorf("panic のログ = %v", panicked)
	}
	if stack, ok := panicked["stack"].(string); !ok || stack == "" {
		t.Errorf("stack = %v", panicked["stack"])
	}
	if access["msg"] != "request" || access["request_id"] != id ||
		access["status"] != 500.0 {
		t.Errorf("アクセスログ = %v", access)
	}
}

// http.ErrAbortHandler は net/http に応答の打ち切りを伝える値なので、
// Recovery で止めない。
func TestMiddleware_AbortHandlerIsNotRecovered(t *testing.T) {
	s := stubUseCases{place: func(
		application.PlaceBookingInput,
	) (application.PlaceBookingOutput, error) {
		panic(http.ErrAbortHandler)
	}}
	var buf bytes.Buffer
	defer func() {
		if r := recover(); r != http.ErrAbortHandler {
			t.Errorf("recover = %v", r)
		}
	}()
	postValid(s, &buf)
	t.Error("panic が伝わらなかった")
}

func TestMiddleware_AccessLogHasRequestID(t *testing.T) {
	s := stubUseCases{cancel: func(
		application.CancelBookingInput,
	) (application.CancelBookingOutput, error) {
		return application.CancelBookingOutput{}, nil
	}}
	var buf bytes.Buffer
	rec := do(newRouter(s, &buf), "POST", "/bookings/b1/cancel", "")
	records := logRecords(t, &buf)
	if len(records) != 1 {
		t.Fatalf("records = %v", records)
	}
	got := records[0]
	if got["request_id"] != rec.Header().Get("X-Request-ID") ||
		got["method"] != "POST" ||
		got["path"] != "/bookings/b1/cancel" ||
		got["status"] != 200.0 {
		t.Errorf("record = %v", got)
	}
}

// 本物のユースケースと in-memory で HTTP から通して確かめる。
func TestRouter_WithUseCases(t *testing.T) {
	repo := memory.NewBookingRepository()
	outbox := memory.NewOutboxWriter()
	tx := memtx.NewTransactor()
	clock := fixedClock{now}
	fee, err := domain.NewMoney(3000)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	price, err := domain.NewMoney(300)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	terms := application.LessonTerms{
		Start:    time.Date(2030, time.June, 1, 10, 0, 0, 0, jst),
		Duration: time.Hour,
		Fee:      fee,
		Location: jst,
		Options: []application.OfferedOption{
			{Name: "シューズ", UnitPrice: price},
		},
	}
	var buf bytes.Buffer
	router := presentation.NewRouter(presentation.Deps{
		Place: application.NewPlaceBooking(
			repo, memory.NewBookingQueries(repo), outbox, tx, clock,
			fixed.NewCoupons(), terms),
		Cancel: application.NewCancelBooking(
			repo, outbox, tx, clock),
		Queries:  memory.NewBookingQueries(repo),
		Logger:   slog.New(slog.NewJSONHandler(&buf, nil)),
		Location: jst,
		Timeout:  time.Second,
	})

	rec := do(router, "POST", "/bookings", validBody)
	if rec.Code != http.StatusCreated {
		t.Fatalf("予約: %d: %s", rec.Code, rec.Body)
	}
	body := decode(t, rec)
	if body["status"] != "pending" {
		t.Fatalf("body = %v", body)
	}
	id, ok := body["id"].(string)
	if !ok {
		t.Fatalf("body = %v", body)
	}

	rec = do(router, "POST", "/bookings", validBody)
	if rec.Code != http.StatusConflict {
		t.Errorf("重なる予約: %d: %s", rec.Code, rec.Body)
	}
	rec = do(router, "POST", "/bookings/"+id+"/cancel", "")
	if rec.Code != http.StatusConflict {
		t.Errorf("pending のキャンセル: %d: %s", rec.Code, rec.Body)
	}
	missing := ids.NewID[ids.BookingMarker]().String()
	rec = do(router, "POST", "/bookings/"+missing+"/cancel", "")
	if rec.Code != http.StatusNotFound {
		t.Errorf("無い予約のキャンセル: %d: %s", rec.Code, rec.Body)
	}
	const member = "0196f5a0-0000-7000-8000-000000000001"
	rec = do(router, "GET", "/members/"+member+"/bookings", "")
	bookings, ok := decode(t, rec)["bookings"].([]any)
	if !ok || len(bookings) != 1 {
		t.Errorf("一覧: %s", rec.Body)
	}
	const options = `"options":[` +
		`{"name":"シューズ","quantity":2,"subtotal_yen":600}]`
	if !strings.Contains(rec.Body.String(), options) {
		t.Errorf("一覧の明細: %s", rec.Body)
	}

	// 別の会員なら同じ時間帯も予約できる
	const other = "0196f5a0-0000-7000-8000-000000000003"
	withCoupon := func(code string) string {
		body := strings.Replace(validBody, member, other, 1)
		return strings.Replace(
			body, "{", `{"coupon_code":"`+code+`",`, 1)
	}
	rec = do(router, "POST", "/bookings", withCoupon("UNKNOWN"))
	if rec.Code != http.StatusBadRequest ||
		decode(t, rec)["kind"] != "invalid" {
		t.Errorf("不明なコード: %d: %s", rec.Code, rec.Body)
	}
	rec = do(router, "POST", "/bookings", withCoupon("WELCOME10"))
	if rec.Code != http.StatusCreated {
		t.Fatalf("クーポンつきの予約: %d: %s", rec.Code, rec.Body)
	}
	rec = do(router, "GET", "/members/"+other+"/bookings", "")
	bookings, ok = decode(t, rec)["bookings"].([]any)
	if !ok || len(bookings) != 1 {
		t.Fatalf("一覧: %s", rec.Body)
	}
	got, ok := bookings[0].(map[string]any)
	if !ok || got["total_yen"] != 3240.0 {
		t.Errorf("割引後の合計: %s", rec.Body)
	}
}
