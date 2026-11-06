// Package repositorytest は BookingRepository の契約テストである。
// in-memory と PostgreSQL の両方に同じテストを流し、
// errors.Is の結果まで一致させる。
package repositorytest

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"github.com/google/go-cmp/cmp"
)

var jst = time.FixedZone("JST", 9*60*60)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

// 予約はすべて2030年6月で、現在は5月1日とする。
var clock = fixedClock{
	now: time.Date(2030, time.May, 1, 0, 0, 0, 0, jst),
}

// Transactor は契約テストが使う Tx の境界である。domain は
// application を import できないので、使う側のここで宣言する。
type Transactor interface {
	Do(ctx context.Context, fn func(context.Context) error) error
}

// txRepository は Add と Save をそれぞれ1つの Tx で実行する。
type txRepository struct {
	domain.BookingRepository
	tx Transactor
}

func (r txRepository) Add(
	ctx context.Context, b *domain.Booking,
) error {
	return r.tx.Do(ctx, func(ctx context.Context) error {
		return r.BookingRepository.Add(ctx, b)
	})
}

func (r txRepository) Save(
	ctx context.Context, b *domain.Booking,
) error {
	return r.tx.Do(ctx, func(ctx context.Context) error {
		return r.BookingRepository.Save(ctx, b)
	})
}

// Run は契約テストを実行する。newRepo はテストごとに
// 空のリポジトリと、それに対応する Transactor を返す。
func Run(
	t *testing.T,
	newRepo func(
		t *testing.T,
	) (domain.BookingRepository, Transactor),
) {
	t.Helper()
	cases := []struct {
		name string
		run  func(*testing.T, domain.BookingRepository)
	}{
		{"往復で全項目が一致する", testRoundTrip},
		{"明細なしでも往復できる", testRoundTripNoOptions},
		{"Add はイベントを取り出さない", testAddKeepsEvents},
		{"重複した Add は ErrDuplicate", testDuplicateAdd},
		{"無い予約の FindByID は ErrNotFound", testFindMissing},
		{"無い予約の Save は ErrNotFound", testSaveMissing},
		{"2者の競合は後者が ErrVersionConflict", testConflict},
		{"同時の Save は1者だけ成功する", testConcurrentSave},
		{"状態を2回変更しても Save できる", testTwoChanges},
		{"変更なしの Save は成功する", testNoChangeSave},
		{"同じオブジェクトの2度目の Save は競合", testSaveTwice},
		{"読み出した集約を変えても保存値は変わらない", testNoSharing},
		{"Save は明細を入れ替える", testSaveReplacesOptions},
		{"時間帯の重複は ErrOverlap", testOverlap},
		{"重複の判定は会員ごと", testOverlapOtherMember},
		{"接する時間帯は重複しない", testAdjacentSlots},
		{"端数のある終了も同じに判定する", testFractionalEnd},
		{"終わった予約は時間帯を空ける", testEndedFreesSlot},
		{"同時の重複は片方だけ成功する", testConcurrentOverlap},
		{"確定と重なる予約が同時でも重複になる", testConcurrentConfirm},
		{"親と明細を同じ時点で読む", testConsistentRead},
		{"ゼロ値の集約は保存しない", testZeroBooking},
		{"キャンセル済みの context はエラー", testCanceledContext},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			repo, tx := newRepo(t)
			c.run(t, txRepository{repo, tx})
		})
	}
	txCases := []struct {
		name string
		run  func(*testing.T, domain.BookingRepository, Transactor)
	}{
		{"TestAdd_WithoutTransaction", testAddWithoutTransaction},
		{"TestSave_WithoutTransaction", testSaveWithoutTransaction},
		{"Do の中で外側の ctx を渡すと拒否する", testOuterContextInDo},
		{"ゼロ値は Tx の有無より先に拒否する", testZeroWithoutTx},
		{"キャンセルは Tx の有無より先に返す", testCanceledWithoutTx},
		{"Tx の中で ctx をキャンセルすると戻る", testCanceledInDo},
		{"FindByID は Tx の中で書いた値を読める", testFindInTx},
		{"Do が失敗すると Add と Save が戻る", testRollback},
		{"入れ子の Do は外側に参加する", testNestedDo},
		{"Do の panic でも Add が戻る", testPanicRollsBack},
	}
	for _, c := range txCases {
		t.Run(c.name, func(t *testing.T) {
			repo, tx := newRepo(t)
			c.run(t, repo, tx)
		})
	}
}

func money(t *testing.T, yen int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(yen)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

func option(
	t *testing.T, name string, price int64, n int,
) domain.BookingOption {
	t.Helper()
	o, err := domain.NewBookingOption(name, money(t, price), n)
	if err != nil {
		t.Fatalf("NewBookingOption: %v", err)
	}
	return o
}

// bookingAt は会員と日時を指定して1時間の予約を作る。
func bookingAt(
	t *testing.T, member ids.MemberID, day, hour int,
) *domain.Booking {
	t.Helper()
	start := time.Date(2030, time.June, day, hour, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		member,
		ids.NewID[ids.LessonMarker](),
		slot,
		money(t, 3000),
		clock,
	)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	return b
}

func newBooking(t *testing.T) *domain.Booking {
	t.Helper()
	return bookingAt(t, ids.NewID[ids.MemberMarker](), 1, 10)
}

func mustAdd(
	t *testing.T, r domain.BookingRepository, b *domain.Booking,
) {
	t.Helper()
	if err := r.Add(t.Context(), b); err != nil {
		t.Fatalf("Add: %v", err)
	}
}

func mustSave(
	t *testing.T, r domain.BookingRepository, b *domain.Booking,
) {
	t.Helper()
	if err := r.Save(t.Context(), b); err != nil {
		t.Fatalf("Save: %v", err)
	}
}

func mustFind(
	t *testing.T, r domain.BookingRepository, id ids.BookingID,
) *domain.Booking {
	t.Helper()
	b, err := r.FindByID(t.Context(), id)
	if err != nil {
		t.Fatalf("FindByID: %v", err)
	}
	return b
}

func wantErr(t *testing.T, got, want error) {
	t.Helper()
	if !errors.Is(got, want) {
		t.Fatalf("err = %v, want %v", got, want)
	}
}

// fullBooking は明細3件と割引を持つ確定済みの予約を作る。
func fullBooking(t *testing.T) *domain.Booking {
	t.Helper()
	b := newBooking(t)
	for _, o := range []domain.BookingOption{
		option(t, "シューズ", 500, 2),
		option(t, "タオル", 200, 1),
		option(t, "ウェア", 800, 1),
	} {
		if err := b.AddOption(o, clock); err != nil {
			t.Fatalf("AddOption: %v", err)
		}
	}
	if err := b.ApplyDiscount(money(t, 300)); err != nil {
		t.Fatalf("ApplyDiscount: %v", err)
	}
	if err := b.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	return b
}

// #region round-trip

func testRoundTrip(t *testing.T, r domain.BookingRepository) {
	b := fullBooking(t)
	mustAdd(t, r, b)

	got := mustFind(t, r, b.ID())
	if diff := cmp.Diff(b.State(), got.State()); diff != "" {
		t.Fatalf("差分 (-want +got):\n%s", diff)
	}
	if !got.Equal(b) {
		t.Error("Equal が false")
	}
	if !got.Slot().Equal(b.Slot()) {
		t.Errorf("Slot = %v", got.Slot())
	}
	if !got.Total().Equal(b.Total()) {
		t.Errorf("Total = %d", got.Total().Amount())
	}
	if events := got.PullEvents(); len(events) != 0 {
		t.Errorf("読み出しでイベントが増えた: %v", events)
	}
}

// #endregion

func testRoundTripNoOptions(
	t *testing.T, r domain.BookingRepository,
) {
	b := newBooking(t)
	mustAdd(t, r, b)
	got := mustFind(t, r, b.ID())
	if diff := cmp.Diff(b.State(), got.State()); diff != "" {
		t.Fatalf("差分 (-want +got):\n%s", diff)
	}
}

func testAddKeepsEvents(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	mustAdd(t, r, b)
	if events := b.PullEvents(); len(events) != 1 {
		t.Fatalf("events = %v", events)
	}
}

func testDuplicateAdd(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	mustAdd(t, r, b)
	wantErr(t, r.Add(t.Context(), b), domain.ErrDuplicate)

	// 時間帯が別でも ID が同じなら重複である
	st := b.State()
	st.Start = st.Start.AddDate(0, 0, 1)
	st.End = st.End.AddDate(0, 0, 1)
	same, err := domain.ReconstructBooking(st)
	if err != nil {
		t.Fatalf("ReconstructBooking: %v", err)
	}
	wantErr(t, r.Add(t.Context(), same), domain.ErrDuplicate)
}

func testFindMissing(t *testing.T, r domain.BookingRepository) {
	id := ids.NewID[ids.BookingMarker]()
	got, err := r.FindByID(t.Context(), id)
	wantErr(t, err, domain.ErrNotFound)
	if got != nil {
		t.Fatalf("got = %v", got)
	}
}

func testSaveMissing(t *testing.T, r domain.BookingRepository) {
	err := r.Save(t.Context(), newBooking(t))
	wantErr(t, err, domain.ErrNotFound)
}

// #region conflict

func testConflict(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	mustAdd(t, r, b)
	first := mustFind(t, r, b.ID())
	second := mustFind(t, r, b.ID())

	if err := first.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	mustSave(t, r, first)

	opt := option(t, "シューズ", 500, 1)
	if err := second.AddOption(opt, clock); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	err := r.Save(t.Context(), second)
	wantErr(t, err, domain.ErrVersionConflict)

	// 版の判定で拒否された側の変更は何も残らない
	got := mustFind(t, r, b.ID())
	if got.Status() != domain.StatusConfirmed {
		t.Errorf("Status = %s", got.Status())
	}
	if len(got.Options()) != 0 {
		t.Errorf("Options = %v", got.Options())
	}
}

// #endregion

// #region concurrent-save

// 同じ版を読んだ2者が同時に保存すると、1者だけが成功する。
func testConcurrentSave(
	t *testing.T, r domain.BookingRepository,
) {
	const rounds = 20
	for round := range rounds {
		b := bookingAt(t, ids.NewID[ids.MemberMarker](), 1, 10)
		mustAdd(t, r, b)
		first := mustFind(t, r, b.ID())
		second := mustFind(t, r, b.ID())
		if err := first.Confirm(); err != nil {
			t.Fatalf("Confirm: %v", err)
		}
		if err := second.Reject(); err != nil {
			t.Fatalf("Reject: %v", err)
		}
		errs := make([]error, 2)
		var done sync.WaitGroup
		start := make(chan struct{})
		for i, loaded := range []*domain.Booking{first, second} {
			done.Go(func() {
				<-start
				errs[i] = r.Save(t.Context(), loaded)
			})
		}
		close(start)
		done.Wait()
		checkOneWins(t, round, errs)
	}
}

// #endregion

func checkOneWins(t *testing.T, round int, errs []error) {
	t.Helper()
	ok, conflict := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case errors.Is(err, domain.ErrVersionConflict):
			conflict++
		default:
			t.Fatalf("round %d: 想定外のエラー: %v", round, err)
		}
	}
	if ok != 1 || conflict != 1 {
		t.Fatalf("round %d: 成功 %d 件, 競合 %d 件",
			round, ok, conflict)
	}
}

// #region two-changes

func testTwoChanges(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	mustAdd(t, r, b)
	loaded := mustFind(t, r, b.ID())
	if err := loaded.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	if err := loaded.Cancel(clock); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	mustSave(t, r, loaded)

	got := mustFind(t, r, b.ID())
	if got.Status() != domain.StatusCancelled {
		t.Errorf("Status = %s", got.Status())
	}
	if got.Version() != loaded.Version()+1 {
		t.Errorf("Version = %d", got.Version())
	}
}

// #endregion

// #region no-change

func testNoChangeSave(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	mustAdd(t, r, b)
	loaded := mustFind(t, r, b.ID())
	mustSave(t, r, loaded)
	got := mustFind(t, r, b.ID())
	if got.Version() != loaded.Version()+1 {
		t.Errorf("Version = %d", got.Version())
	}
}

// #endregion

// #region save-twice

func testSaveTwice(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	mustAdd(t, r, b)
	loaded := mustFind(t, r, b.ID())
	mustSave(t, r, loaded)
	// Save は loaded の版を変えない。使い続けるときは読み直す
	err := r.Save(t.Context(), loaded)
	wantErr(t, err, domain.ErrVersionConflict)
	mustSave(t, r, mustFind(t, r, b.ID()))
}

// #endregion

func testNoSharing(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	opt := option(t, "シューズ", 500, 1)
	if err := b.AddOption(opt, clock); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	mustAdd(t, r, b)
	want := mustFind(t, r, b.ID()).State()

	// Add に渡した集約を後から変える
	other := option(t, "タオル", 200, 1)
	if err := b.AddOption(other, clock); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	if err := b.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	// 読み出した集約を保存せずに変える
	loaded := mustFind(t, r, b.ID())
	if err := loaded.AddOption(other, clock); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	if err := loaded.Reject(); err != nil {
		t.Fatalf("Reject: %v", err)
	}

	got := mustFind(t, r, b.ID()).State()
	if diff := cmp.Diff(want, got); diff != "" {
		t.Fatalf("差分 (-want +got):\n%s", diff)
	}
}

// withOptions は明細だけを入れ替えた集約を返す。集約に明細を外す
// 操作は無いので、Save が明細を置き換えることは再構築で確かめる。
func withOptions(
	t *testing.T, b *domain.Booking, options ...domain.OptionState,
) *domain.Booking {
	t.Helper()
	st := b.State()
	st.Options = options
	got, err := domain.ReconstructBooking(st)
	if err != nil {
		t.Fatalf("ReconstructBooking: %v", err)
	}
	return got
}

func testSaveReplacesOptions(
	t *testing.T, r domain.BookingRepository,
) {
	b := newBooking(t)
	for _, o := range []domain.BookingOption{
		option(t, "a", 100, 1),
		option(t, "b", 200, 2),
		option(t, "c", 300, 3),
	} {
		if err := b.AddOption(o, clock); err != nil {
			t.Fatalf("AddOption: %v", err)
		}
	}
	mustAdd(t, r, b)

	// 外した名前を付け直しても一意の制約に反しない
	loaded := mustFind(t, r, b.ID())
	current := loaded.State().Options
	loaded = withOptions(t, loaded, current[1], current[2],
		domain.OptionState{
			Name: "a", UnitPriceYen: 150, Quantity: 1,
		})
	mustSave(t, r, loaded)

	got := mustFind(t, r, b.ID())
	want := loaded.State().Options
	diff := cmp.Diff(want, got.State().Options)
	if diff != "" {
		t.Fatalf("順序を含めて一致しない (-want +got):\n%s", diff)
	}
}

func testOverlap(t *testing.T, r domain.BookingRepository) {
	member := ids.NewID[ids.MemberMarker]()
	mustAdd(t, r, bookingAt(t, member, 1, 10))
	err := r.Add(t.Context(), bookingAt(t, member, 1, 10))
	wantErr(t, err, domain.ErrOverlap)

	// 別の予約 ID・別のレッスンでも、時間が重なれば拒否する
	start := time.Date(2030, time.June, 1, 9, 0, 0, 0, jst)
	slot, err := domain.NewTimeSlot(start, 2*time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	longer, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](), member,
		ids.NewID[ids.LessonMarker](), slot,
		money(t, 3000), clock,
	)
	if err != nil {
		t.Fatalf("NewBooking: %v", err)
	}
	wantErr(t, r.Add(t.Context(), longer), domain.ErrOverlap)
}

func testOverlapOtherMember(
	t *testing.T, r domain.BookingRepository,
) {
	mustAdd(t, r, bookingAt(
		t, ids.NewID[ids.MemberMarker](), 1, 10))
	mustAdd(t, r, bookingAt(
		t, ids.NewID[ids.MemberMarker](), 1, 10))
}

func testAdjacentSlots(t *testing.T, r domain.BookingRepository) {
	member := ids.NewID[ids.MemberMarker]()
	mustAdd(t, r, bookingAt(t, member, 1, 10))
	mustAdd(t, r, bookingAt(t, member, 1, 9))
	mustAdd(t, r, bookingAt(t, member, 1, 11))
	mustAdd(t, r, bookingAt(t, member, 2, 10))
}

// 終了が1マイクロ秒だけ次の時間帯に入る予約は往復でき、重なる。
// 1マイクロ秒より細かい値は NewTimeSlot と再構築が拒否するので、
// 実装のあいだで丸めの差は出ない。
func testFractionalEnd(t *testing.T, r domain.BookingRepository) {
	member := ids.NewID[ids.MemberMarker]()
	st := bookingAt(t, member, 1, 10).State()
	st.End = st.End.Add(time.Microsecond)
	b, err := domain.ReconstructBooking(st)
	if err != nil {
		t.Fatalf("ReconstructBooking: %v", err)
	}
	mustAdd(t, r, b)
	got := mustFind(t, r, b.ID()).State()
	if diff := cmp.Diff(st, got); diff != "" {
		t.Fatalf("差分 (-want +got):\n%s", diff)
	}
	err = r.Add(t.Context(), bookingAt(t, member, 1, 11))
	wantErr(t, err, domain.ErrOverlap)
}

func testEndedFreesSlot(t *testing.T, r domain.BookingRepository) {
	member := ids.NewID[ids.MemberMarker]()

	rejected := bookingAt(t, member, 1, 10)
	mustAdd(t, r, rejected)
	loaded := mustFind(t, r, rejected.ID())
	if err := loaded.Reject(); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	mustSave(t, r, loaded)

	cancelled := bookingAt(t, member, 1, 10)
	mustAdd(t, r, cancelled)
	loaded = mustFind(t, r, cancelled.ID())
	if err := loaded.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	mustSave(t, r, loaded)
	wantErr(t, r.Add(t.Context(), bookingAt(t, member, 1, 10)),
		domain.ErrOverlap)
	loaded = mustFind(t, r, cancelled.ID())
	if err := loaded.Cancel(clock); err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	mustSave(t, r, loaded)

	mustAdd(t, r, bookingAt(t, member, 1, 10))
}

// 同じ会員の重なる予約を同時に保存しても、1件だけが成功する。
func testConcurrentOverlap(
	t *testing.T, r domain.BookingRepository,
) {
	const rounds = 20
	for round := range rounds {
		member := ids.NewID[ids.MemberMarker]()
		errs := make([]error, 2)
		var ready, done sync.WaitGroup
		start := make(chan struct{})
		for i := range errs {
			b := bookingAt(t, member, 1+round%28, 10)
			ready.Add(1)
			done.Add(1)
			go func() {
				defer done.Done()
				ready.Done()
				<-start
				errs[i] = r.Add(t.Context(), b)
			}()
		}
		ready.Wait()
		close(start)
		done.Wait()

		ok, overlap := 0, 0
		for _, err := range errs {
			switch {
			case err == nil:
				ok++
			case errors.Is(err, domain.ErrOverlap):
				overlap++
			default:
				t.Fatalf("round %d: 想定外のエラー: %v", round, err)
			}
		}
		if ok != 1 || overlap != 1 {
			t.Fatalf("round %d: 成功 %d 件, 重複 %d 件",
				round, ok, overlap)
		}
	}
}

// 予約の確定と、それに重なる予約の保存が同時に走っても、
// 確定は成功し、重なる予約は ErrOverlap になる。
func testConcurrentConfirm(
	t *testing.T, r domain.BookingRepository,
) {
	const rounds = 20
	for round := range rounds {
		member := ids.NewID[ids.MemberMarker]()
		day := 1 + round%28
		first := bookingAt(t, member, day, 10)
		mustAdd(t, r, first)
		loaded := mustFind(t, r, first.ID())
		if err := loaded.Confirm(); err != nil {
			t.Fatalf("Confirm: %v", err)
		}
		second := bookingAt(t, member, day, 10)

		var saveErr, addErr error
		var done sync.WaitGroup
		start := make(chan struct{})
		done.Go(func() {
			<-start
			saveErr = r.Save(t.Context(), loaded)
		})
		done.Go(func() {
			<-start
			addErr = r.Add(t.Context(), second)
		})
		close(start)
		done.Wait()

		if saveErr != nil {
			t.Fatalf("round %d: Save: %v", round, saveErr)
		}
		if !errors.Is(addErr, domain.ErrOverlap) {
			t.Fatalf("round %d: Add: %v", round, addErr)
		}
	}
}

// 版 v の予約は明細 opt-v だけを持つように更新し続ける。
// 親と明細を別の時点で読むと、この組み合わせが成立しない。
func testConsistentRead(t *testing.T, r domain.BookingRepository) {
	const writes = 100
	b := newBooking(t)
	mustAdd(t, r, b)

	stop := make(chan struct{})
	read := make(chan struct{}, 1)
	done := make(chan struct{})
	defer func() {
		close(stop)
		<-done
	}()
	go func() {
		defer close(done)
		for {
			select {
			case <-stop:
				return
			default:
			}
			got, err := r.FindByID(t.Context(), b.ID())
			if err != nil {
				t.Errorf("FindByID: %v", err)
				return
			}
			if msg := checkPair(got.State()); msg != "" {
				t.Error(msg)
				return
			}
			select {
			case read <- struct{}{}:
			default:
			}
		}
	}()

	for v := range writes {
		// 読み取りが動いていることを確かめてから更新する。
		// 読み取りが1回も走らないまま通るのを防ぐ
		select {
		case <-read:
		case <-done:
			return
		}
		next := domain.OptionState{
			Name:         fmt.Sprintf("opt-%d", v+1),
			UnitPriceYen: 100,
			Quantity:     1,
		}
		mustSave(t, r, withOptions(
			t, mustFind(t, r, b.ID()), next))
	}
}

func checkPair(s domain.BookingState) string {
	if s.Version == 0 {
		if len(s.Options) != 0 {
			return fmt.Sprintf("版0に明細がある: %v", s.Options)
		}
		return ""
	}
	want := fmt.Sprintf("opt-%d", s.Version)
	if len(s.Options) != 1 || s.Options[0].Name != want {
		return fmt.Sprintf("版 %d に明細 %v", s.Version, s.Options)
	}
	return ""
}

// #region add-without-tx

func testAddWithoutTransaction(
	t *testing.T, r domain.BookingRepository, _ Transactor,
) {
	b := newBooking(t)
	wantErr(t, r.Add(t.Context(), b), txn.ErrNoTransaction)
	_, err := r.FindByID(t.Context(), b.ID())
	wantErr(t, err, domain.ErrNotFound)
}

// #endregion

func testSaveWithoutTransaction(
	t *testing.T, r domain.BookingRepository, tx Transactor,
) {
	b := newBooking(t)
	mustAdd(t, txRepository{r, tx}, b)
	loaded := mustFind(t, r, b.ID())
	if err := loaded.Confirm(); err != nil {
		t.Fatalf("Confirm: %v", err)
	}
	err := r.Save(t.Context(), loaded)
	wantErr(t, err, txn.ErrNoTransaction)

	// FindByID は Tx なしで読める。値は変わっていない
	got := mustFind(t, r, b.ID())
	if got.Status() != domain.StatusPending || got.Version() != 0 {
		t.Errorf("保存された: %s v%d", got.Status(), got.Version())
	}
}

// #region outer-ctx

// 外側の ctx を渡す誤りで別にコミットされないことを確かめる。
func testOuterContextInDo(
	t *testing.T, r domain.BookingRepository, tx Transactor,
) {
	outer := t.Context()
	b := newBooking(t)
	err := tx.Do(outer, func(context.Context) error {
		return r.Add(outer, b)
	})
	wantErr(t, err, txn.ErrNoTransaction)
	_, err = r.FindByID(outer, b.ID())
	wantErr(t, err, domain.ErrNotFound)
}

// #endregion

func testZeroWithoutTx(
	t *testing.T, r domain.BookingRepository, _ Transactor,
) {
	wantErr(t, r.Add(t.Context(), &domain.Booking{}),
		domain.ErrInvalidBooking)
	wantErr(t, r.Save(t.Context(), &domain.Booking{}),
		domain.ErrInvalidBooking)
}

func testCanceledWithoutTx(
	t *testing.T, r domain.BookingRepository, _ Transactor,
) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	wantErr(t, r.Add(ctx, newBooking(t)), context.Canceled)
	wantErr(t, r.Save(ctx, newBooking(t)), context.Canceled)
}

func testCanceledInDo(
	t *testing.T, r domain.BookingRepository, tx Transactor,
) {
	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()
	first, second := newBooking(t), newBooking(t)
	err := tx.Do(ctx, func(ctx context.Context) error {
		if err := r.Add(ctx, first); err != nil {
			return err
		}
		cancel()
		return r.Add(ctx, second)
	})
	wantErr(t, err, context.Canceled)
	_, err = r.FindByID(t.Context(), first.ID())
	wantErr(t, err, domain.ErrNotFound)
}

func testFindInTx(
	t *testing.T, r domain.BookingRepository, tx Transactor,
) {
	b := newBooking(t)
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		if err := r.Add(ctx, b); err != nil {
			return err
		}
		got, err := r.FindByID(ctx, b.ID())
		if err != nil {
			return err
		}
		if !got.Equal(b) {
			t.Errorf("FindByID = %v", got.ID())
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
}

// addAndConfirm は added を保存し、id の予約を確定して保存する。
func addAndConfirm(
	ctx context.Context, r domain.BookingRepository,
	added *domain.Booking, id ids.BookingID,
) error {
	if err := r.Add(ctx, added); err != nil {
		return err
	}
	loaded, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := loaded.Confirm(); err != nil {
		return err
	}
	return r.Save(ctx, loaded)
}

// #region rollback

func testRollback(
	t *testing.T, r domain.BookingRepository, tx Transactor,
) {
	member := ids.NewID[ids.MemberMarker]()
	saved := bookingAt(t, member, 1, 10)
	mustAdd(t, txRepository{r, tx}, saved)
	added := bookingAt(t, member, 2, 10)

	boom := errors.New("boom")
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		err := addAndConfirm(ctx, r, added, saved.ID())
		if err != nil {
			return err
		}
		return boom
	})
	wantErr(t, err, boom)

	_, err = r.FindByID(t.Context(), added.ID())
	wantErr(t, err, domain.ErrNotFound)
	got := mustFind(t, r, saved.ID())
	if got.Status() != domain.StatusPending || got.Version() != 0 {
		t.Errorf("Save が残った: %s v%d",
			got.Status(), got.Version())
	}
}

// #endregion

func testNestedDo(
	t *testing.T, r domain.BookingRepository, tx Transactor,
) {
	b := newBooking(t)
	boom := errors.New("boom")
	err := tx.Do(t.Context(), func(ctx context.Context) error {
		err := tx.Do(ctx, func(ctx context.Context) error {
			return r.Add(ctx, b)
		})
		if err != nil {
			return err
		}
		return boom
	})
	wantErr(t, err, boom)
	_, err = r.FindByID(t.Context(), b.ID())
	wantErr(t, err, domain.ErrNotFound)
}

func testPanicRollsBack(
	t *testing.T, r domain.BookingRepository, tx Transactor,
) {
	b := newBooking(t)
	func() {
		defer func() {
			if recover() == nil {
				t.Error("panic が伝わらなかった")
			}
		}()
		_ = tx.Do(t.Context(), func(ctx context.Context) error {
			if err := r.Add(ctx, b); err != nil {
				return err
			}
			panic("boom")
		})
	}()
	_, err := r.FindByID(t.Context(), b.ID())
	wantErr(t, err, domain.ErrNotFound)
}

// #region zero

func testZeroBooking(t *testing.T, r domain.BookingRepository) {
	wantErr(t, r.Add(t.Context(), &domain.Booking{}),
		domain.ErrInvalidBooking)
	wantErr(t, r.Save(t.Context(), &domain.Booking{}),
		domain.ErrInvalidBooking)
}

// #endregion

func testCanceledContext(t *testing.T, r domain.BookingRepository) {
	b := newBooking(t)
	mustAdd(t, r, b)
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	wantErr(t, r.Add(ctx, newBooking(t)), context.Canceled)
	wantErr(t, r.Save(ctx, b), context.Canceled)
	_, err := r.FindByID(ctx, b.ID())
	wantErr(t, err, context.Canceled)
}
