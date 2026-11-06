package application_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/application"
	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
)

func newPlace(
	t *testing.T, d memoryDeps,
) *application.PlaceBooking {
	t.Helper()
	return application.NewPlaceBooking(
		d.repo, d.queries, d.outbox, d.tx, fixedClock{now},
		d.coupons, lessons(t))
}

func TestPlaceBooking_SavesPendingAndRecordsEvent(t *testing.T) {
	d := newMemoryDeps()
	in := placeInput()
	out, err := newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if out.Status != "pending" {
		t.Errorf("Status = %s", out.Status)
	}
	id, err := ids.ParseID[ids.BookingMarker](out.BookingID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	got, err := d.repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	if got.Status() != domain.StatusPending {
		t.Errorf("保存された Status = %s", got.Status())
	}
	if got.Total().Amount() != 3000 {
		t.Errorf("Total = %d", got.Total().Amount())
	}
	if !got.Slot().Start().Equal(lessonStart) ||
		!got.Slot().End().Equal(lessonStart.Add(time.Hour)) {
		t.Errorf("Slot = %+v", got.Slot())
	}
	events := d.outbox.Events()
	if len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
	placed, ok := events[0].(domain.BookingPlaced)
	if !ok || !placed.BookingID.Equal(id) {
		t.Errorf("event = %#v", events[0])
	}
}

func TestPlaceBooking_InvalidMemberIDIsKindInvalid(t *testing.T) {
	in := placeInput()
	in.MemberID = "not-a-uuid"
	_, err := newPlace(t, newMemoryDeps()).Execute(t.Context(), in)
	got := wantKind(t, err, application.KindInvalid)
	if !errors.Is(err, ids.ErrInvalidID) {
		t.Errorf("原因をたどれない: %v", got)
	}
}

func TestPlaceBooking_InvalidLessonIDIsKindInvalid(t *testing.T) {
	in := placeInput()
	in.LessonID = ""
	_, err := newPlace(t, newMemoryDeps()).Execute(t.Context(), in)
	wantKind(t, err, application.KindInvalid)
	if !errors.Is(err, ids.ErrInvalidID) {
		t.Errorf("err = %v", err)
	}
}

// レッスンの条件が予約できないものなら、入力の誤りとして返す。
func TestPlaceBooking_InvalidLessonTermsIsKindInvalid(t *testing.T) {
	cases := []struct {
		name   string
		change func(*application.LessonTerms)
		cause  error
	}{
		{"開始が無い", func(l *application.LessonTerms) {
			l.Start = time.Time{}
		}, domain.ErrInvalidTimeSlot},
		{"長さが0", func(l *application.LessonTerms) {
			l.Duration = 0
		}, domain.ErrInvalidDuration},
		{"正時でない", func(l *application.LessonTerms) {
			l.Start = l.Start.Add(time.Minute)
		}, domain.ErrNotOnTheHour},
		{"営業時間の外", func(l *application.LessonTerms) {
			l.Start = time.Date(2030, time.June, 1, 8, 0, 0, 0, jst)
		}, domain.ErrOutOfBusinessHours},
		{"過去の開始", func(l *application.LessonTerms) {
			l.Start = time.Date(
				2030, time.April, 1, 10, 0, 0, 0, jst)
		}, domain.ErrStartInPast},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lesson := terms(t)
			c.change(&lesson)
			d := newMemoryDeps()
			place := application.NewPlaceBooking(
				d.repo, d.queries, d.outbox, d.tx, fixedClock{now},
				d.coupons, stubLessons{terms: lesson})
			_, err := place.Execute(t.Context(), placeInput())
			wantKind(t, err, application.KindInvalid)
			if !errors.Is(err, c.cause) {
				t.Errorf("err = %v, want %v", err, c.cause)
			}
			if len(d.outbox.Events()) != 0 {
				t.Errorf("events = %v", d.outbox.Events())
			}
		})
	}
}

func TestPlaceBooking_UnknownLessonIsKindInvalid(t *testing.T) {
	d := newMemoryDeps()
	place := application.NewPlaceBooking(
		d.repo, d.queries, d.outbox, d.tx, fixedClock{now},
		d.coupons, stubLessons{err: application.ErrLessonNotFound})
	_, err := place.Execute(t.Context(), placeInput())
	wantKind(t, err, application.KindInvalid)
	if !errors.Is(err, application.ErrLessonNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestPlaceBooking_CatalogFailureIsKindInternal(t *testing.T) {
	boom := errors.New("catalog")
	d := newMemoryDeps()
	place := application.NewPlaceBooking(
		d.repo, d.queries, d.outbox, d.tx, fixedClock{now},
		d.coupons, stubLessons{err: boom})
	_, err := place.Execute(t.Context(), placeInput())
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

// 営業時間の判定に使うのは LessonTerms の Location である。
func TestPlaceBooking_UsesLocationFromTerms(t *testing.T) {
	d := newMemoryDeps()
	lesson := terms(t)
	// UTC の10時は JST の19時で営業時間内、
	// UTC の13時は JST の22時で営業時間外になる。
	lesson.Start = time.Date(
		2030, time.June, 1, 10, 0, 0, 0, time.UTC)
	place := application.NewPlaceBooking(
		d.repo, d.queries, d.outbox, d.tx, fixedClock{now},
		d.coupons, stubLessons{terms: lesson})
	_, err := place.Execute(t.Context(), placeInput())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	lesson.Start = time.Date(
		2030, time.June, 1, 13, 0, 0, 0, time.UTC)
	place = application.NewPlaceBooking(
		d.repo, d.queries, d.outbox, d.tx, fixedClock{now},
		d.coupons, stubLessons{terms: lesson})
	_, err = place.Execute(t.Context(), placeInput())
	if !errors.Is(err, domain.ErrOutOfBusinessHours) {
		t.Fatalf("err = %v", err)
	}
}

// 単価はレッスンの値を使う。入力で価格は指定できない。
func TestPlaceBooking_OptionsUseLessonPrices(t *testing.T) {
	d := newMemoryDeps()
	in := placeInput()
	in.Options = []application.OptionInput{
		{Name: "シューズ", Quantity: 2},
		{Name: "タオル", Quantity: 1},
	}
	out, err := newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	id, err := ids.ParseID[ids.BookingMarker](out.BookingID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	got, err := d.repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	// 3000 + 300*2 + 100
	if got.Total().Amount() != 3700 {
		t.Errorf("Total = %d", got.Total().Amount())
	}
	if len(got.Options()) != 2 {
		t.Errorf("Options = %v", got.Options())
	}
}

func TestPlaceBooking_InvalidOptionsAreKindInvalid(t *testing.T) {
	shoes := application.OptionInput{Name: "シューズ", Quantity: 1}
	many := make([]application.OfferedOption, 6)
	input := make([]application.OptionInput, 6)
	for i := range many {
		name := fmt.Sprintf("オプション%d", i)
		many[i] = application.OfferedOption{Name: name}
		input[i] = application.OptionInput{Name: name, Quantity: 1}
	}
	cases := []struct {
		name    string
		options []application.OptionInput
		offered []application.OfferedOption
		cause   error
	}{
		{name: "レッスンに無いオプション",
			options: []application.OptionInput{
				{Name: "ヨガマット", Quantity: 1},
			},
			cause: application.ErrUnknownOption},
		{name: "数量が0",
			options: []application.OptionInput{
				{Name: "シューズ", Quantity: 0},
			},
			cause: domain.ErrInvalidOption},
		{name: "同じオプションが2件",
			options: []application.OptionInput{shoes, shoes},
			cause:   domain.ErrDuplicateOption},
		{name: "上限を超える",
			options: input, offered: many,
			cause: domain.ErrOptionLimit},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			lesson := terms(t)
			if c.offered != nil {
				lesson.Options = c.offered
			}
			d := newMemoryDeps()
			place := application.NewPlaceBooking(
				d.repo, d.queries, d.outbox, d.tx, fixedClock{now},
				d.coupons, stubLessons{terms: lesson})
			in := placeInput()
			in.Options = c.options
			_, err := place.Execute(t.Context(), in)
			wantKind(t, err, application.KindInvalid)
			if !errors.Is(err, c.cause) {
				t.Errorf("err = %v, want %v", err, c.cause)
			}
		})
	}
}

// 開始まで24時間を切ると、オプションは付けられない。
func TestPlaceBooking_OptionsTooLateIsKindInvalid(t *testing.T) {
	d := newMemoryDeps()
	late := lessonStart.Add(-23 * time.Hour)
	place := application.NewPlaceBooking(
		d.repo, d.queries, d.outbox, d.tx, fixedClock{late},
		d.coupons, lessons(t))
	in := placeInput()
	in.Options = []application.OptionInput{
		{Name: "シューズ", Quantity: 1},
	}
	_, err := place.Execute(t.Context(), in)
	got := wantKind(t, err, application.KindInvalid)
	if !errors.Is(err, domain.ErrChangeTooLate) {
		t.Errorf("err = %v", err)
	}
	if got.Message != "オプションを追加できる期限を過ぎています" {
		t.Errorf("Message = %s", got.Message)
	}
	// オプションが無ければ同じ時刻でも予約できる
	in.Options = nil
	if _, err := place.Execute(t.Context(), in); err != nil {
		t.Errorf("オプションなし: %v", err)
	}
}

// 数量は入力で受けるので、金額のあふれは入力の誤りである。
func TestPlaceBooking_MoneyOverflowIsKindInvalid(t *testing.T) {
	cases := []struct {
		name     string
		quantity int
		coupon   string
	}{
		{name: "明細の小計", quantity: 1 << 60},
		{name: "割引の計算", quantity: 1 << 54, coupon: "WELCOME10"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := newMemoryDeps()
			in := placeInput()
			in.Options = []application.OptionInput{
				{Name: "シューズ", Quantity: c.quantity},
			}
			in.CouponCode = c.coupon
			_, err := newPlace(t, d).Execute(t.Context(), in)
			wantKind(t, err, application.KindInvalid)
			if !errors.Is(err, domain.ErrMoneyOverflow) {
				t.Errorf("err = %v", err)
			}
		})
	}
}

func TestPlaceBooking_OverlapIsKindConflict(t *testing.T) {
	d := newMemoryDeps()
	place := newPlace(t, d)
	in := placeInput()
	if _, err := place.Execute(t.Context(), in); err != nil {
		t.Fatalf("1件目: %v", err)
	}
	in.LessonID = ids.NewID[ids.LessonMarker]().String()
	_, err := place.Execute(t.Context(), in)
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrOverlap) {
		t.Errorf("err = %v", err)
	}
	if len(d.outbox.Events()) != 1 {
		t.Errorf("events = %v", d.outbox.Events())
	}
}

func TestPlaceBooking_RepositoryFailureKeepsCause(t *testing.T) {
	boom := errors.New("connection refused: secret-host:5432")
	d := newMemoryDeps()
	place := application.NewPlaceBooking(
		failingRepository{d.repo, fmt.Errorf("add: %w", boom)},
		d.queries, d.outbox, d.tx, fixedClock{now}, d.coupons,
		lessons(t))

	_, err := place.Execute(t.Context(), placeInput())
	got := wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Errorf("原因をたどれない: %v", err)
	}
	// Message は外に出す文なので、原因の文字列を含めない
	if strings.Contains(got.Message, "secret-host") {
		t.Errorf("Message に原因が出た: %s", got.Message)
	}
}

// Tx の無い書き込みの誤りは、利用者の入力の誤りとして見せない。
func TestPlaceBooking_NoTransactionIsKindInternal(t *testing.T) {
	d := newMemoryDeps()
	place := application.NewPlaceBooking(
		failingRepository{d.repo, txn.ErrNoTransaction}, d.queries,
		d.outbox, d.tx, fixedClock{now}, d.coupons, lessons(t))
	_, err := place.Execute(t.Context(), placeInput())
	wantKind(t, err, application.KindInternal)
}

func TestPlaceBooking_CanceledContextKeepsCause(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	place := newPlace(t, newMemoryDeps())
	_, err := place.Execute(ctx, placeInput())
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, context.Canceled) {
		t.Errorf("err = %v", err)
	}
}

func TestPlaceBooking_OutboxFailureRollsBack(t *testing.T) {
	d := newMemoryDeps()
	boom := errors.New("outbox")
	place := application.NewPlaceBooking(
		d.repo, d.queries, failingOutbox{boom}, d.tx,
		fixedClock{now}, d.coupons, lessons(t))
	in := placeInput()
	_, err := place.Execute(t.Context(), in)
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
	// 予約が残っていれば、同じ時間帯は重複になる
	_, err = newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Errorf("予約が残った: %v", err)
	}
}

// 2件目はリポジトリに渡す前に止まる。Add まで進めば boom になる。
func TestPlaceBooking_DetectsOverlapBeforeAdd(t *testing.T) {
	d := newMemoryDeps()
	in := placeInput()
	_, err := newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Fatalf("1件目: %v", err)
	}
	boom := errors.New("add")
	place := application.NewPlaceBooking(
		failingRepository{d.repo, boom}, d.queries, d.outbox, d.tx,
		fixedClock{now}, d.coupons, lessons(t))
	in.LessonID = ids.NewID[ids.LessonMarker]().String()
	_, err = place.Execute(t.Context(), in)
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrOverlap) {
		t.Errorf("err = %v", err)
	}
}

// 読み取りで見えなかった予約は、リポジトリが ErrOverlap で止める。
func TestPlaceBooking_StaleReadIsStoppedByRepository(t *testing.T) {
	d := newMemoryDeps()
	d.queries = staleQueries{d.queries}
	place := newPlace(t, d)
	in := placeInput()
	if _, err := place.Execute(t.Context(), in); err != nil {
		t.Fatalf("1件目: %v", err)
	}
	in.LessonID = ids.NewID[ids.LessonMarker]().String()
	_, err := place.Execute(t.Context(), in)
	wantKind(t, err, application.KindConflict)
	if !errors.Is(err, domain.ErrOverlap) {
		t.Errorf("err = %v", err)
	}
	if len(d.outbox.Events()) != 1 {
		t.Errorf("events = %v", d.outbox.Events())
	}
}

func TestPlaceBooking_QueryFailureIsKindInternal(t *testing.T) {
	d := newMemoryDeps()
	boom := errors.New("query")
	d.queries = failingQueries{d.queries, boom}
	_, err := newPlace(t, d).Execute(t.Context(), placeInput())
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
	if len(d.outbox.Events()) != 0 {
		t.Errorf("events = %v", d.outbox.Events())
	}
}

// findBooking は結果の ID で保存された予約を読む。
func findBooking(
	t *testing.T, d memoryDeps, out application.PlaceBookingOutput,
) *domain.Booking {
	t.Helper()
	id, err := ids.ParseID[ids.BookingMarker](out.BookingID)
	if err != nil {
		t.Fatalf("ParseID: %v", err)
	}
	b, err := d.repo.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	return b
}

// 割合は入力で受け取らず、コードから引いた Coupon で計算する。
func TestPlaceBooking_CouponDiscountsTotal(t *testing.T) {
	d := newMemoryDeps()
	in := placeInput()
	in.CouponCode = "WELCOME10"
	out, err := newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := findBooking(t, d, out).Total().Amount()
	if got != 2700 {
		t.Errorf("Total = %d", got)
	}
}

// コードが無ければ Coupons を呼ばない。呼べば boom になる。
func TestPlaceBooking_NoCouponCodeSkipsLookup(t *testing.T) {
	d := newMemoryDeps()
	d.coupons = failingCoupons{errors.New("boom")}
	out, err := newPlace(t, d).Execute(t.Context(), placeInput())
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	got := findBooking(t, d, out).Total().Amount()
	if got != 3000 {
		t.Errorf("Total = %d", got)
	}
}

func TestPlaceBooking_UnknownCouponIsKindInvalid(t *testing.T) {
	d := newMemoryDeps()
	in := placeInput()
	in.CouponCode = "UNKNOWN"
	_, err := newPlace(t, d).Execute(t.Context(), in)
	wantKind(t, err, application.KindInvalid)
	if !errors.Is(err, application.ErrCouponNotFound) {
		t.Errorf("err = %v", err)
	}
	if len(d.outbox.Events()) != 0 {
		t.Errorf("events = %v", d.outbox.Events())
	}
	// 拒否した予約は残らないので、同じ時間帯をもう一度予約できる
	in.CouponCode = ""
	_, err = newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Errorf("予約が残った: %v", err)
	}
}

func TestPlaceBooking_CouponFailureIsKindInternal(t *testing.T) {
	d := newMemoryDeps()
	boom := errors.New("coupons")
	d.coupons = failingCoupons{boom}
	in := placeInput()
	in.CouponCode = "WELCOME10"
	_, err := newPlace(t, d).Execute(t.Context(), in)
	wantKind(t, err, application.KindInternal)
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

// #region coupon-options

// 割引の対象は料金と明細の合計である。
func TestPlaceBooking_CouponDiscountsOptionsToo(t *testing.T) {
	d := newMemoryDeps()
	in := placeInput()
	in.CouponCode = "WELCOME10"
	in.Options = []application.OptionInput{
		{Name: "シューズ", Quantity: 2},
	}
	out, err := newPlace(t, d).Execute(t.Context(), in)
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	// 3000 + 300*2 = 3600 の10%を引く
	got := findBooking(t, d, out).Total().Amount()
	if got != 3240 {
		t.Errorf("Total = %d", got)
	}
}

// #endregion
