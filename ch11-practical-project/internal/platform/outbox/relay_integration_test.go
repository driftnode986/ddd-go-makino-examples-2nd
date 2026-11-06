//go:build integration

package outbox_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"slices"
	"strings"
	"testing"
	"time"
	"uuid"

	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/google/go-cmp/cmp"
	"github.com/jackc/pgx/v5/pgxpool"
)

const eventType = "test.happened"

type fixture struct {
	pool  *pgxpool.Pool
	store *outbox.Store
	tx    *pgtx.Transactor
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	pool := dbtest.New(t, outbox.Migrations)
	return fixture{
		pool:  pool,
		store: outbox.NewStore(pool),
		tx:    pgtx.NewTransactor(pool),
	}
}

func message(aggregate, name string) outbox.Message {
	return outbox.Message{
		EventID:       outbox.NewEventID(),
		AggregateType: "test",
		AggregateID:   aggregate,
		EventType:     eventType,
		Payload:       fmt.Appendf(nil, `{"name":%q}`, name),
		OccurredAt:    time.Now(),
	}
}

func (f fixture) append(
	t *testing.T, messages ...outbox.Message,
) {
	t.Helper()
	err := f.tx.Do(t.Context(), func(ctx context.Context) error {
		return f.store.Append(ctx, messages...)
	})
	if err != nil {
		t.Fatalf("Append: %v", err)
	}
}

func nameOf(t *testing.T, d outbox.Delivery) string {
	t.Helper()
	var e event
	if err := json.Unmarshal(d.Payload, &e); err != nil {
		t.Fatalf("payload: %v", err)
	}
	return e.Name
}

func TestAppend_RequiresTransaction(t *testing.T) {
	f := newFixture(t)
	err := f.store.Append(t.Context(),
		message(uuid.New().String(), "a"))
	if !errors.Is(err, txn.ErrNoTransaction) {
		t.Fatalf("err = %v", err)
	}
}

func TestAppend_RollsBackWithTransaction(t *testing.T) {
	f := newFixture(t)
	boom := errors.New("boom")
	err := f.tx.Do(t.Context(), func(ctx context.Context) error {
		err := f.store.Append(ctx,
			message(uuid.New().String(), "a"))
		if err != nil {
			return err
		}
		return boom
	})
	if !errors.Is(err, boom) {
		t.Fatalf("err = %v", err)
	}
	got, err := f.store.Deliverable(t.Context(), 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("got = %v, err = %v", got, err)
	}
}

// 連番は集約ごとに1から振られ、未配信の先頭だけが候補に出る。
func TestDeliverable_ReturnsHeadOfEachAggregate(t *testing.T) {
	f := newFixture(t)
	a, b := uuid.New().String(), uuid.New().String()
	f.append(t, message(a, "a1"), message(a, "a2"),
		message(b, "b1"))

	got, err := f.store.Deliverable(t.Context(), 10)
	if err != nil {
		t.Fatalf("Deliverable: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got = %+v", got)
	}
	if nameOf(t, got[0]) != "a1" || got[0].Sequence != 1 ||
		nameOf(t, got[1]) != "b1" || got[1].Sequence != 1 {
		t.Errorf("got = %+v", got)
	}

	err = f.store.MarkPublished(t.Context(), got[0].ID)
	if err != nil {
		t.Fatalf("MarkPublished: %v", err)
	}
	got, err = f.store.Deliverable(t.Context(), 10)
	if err != nil {
		t.Fatalf("Deliverable: %v", err)
	}
	if nameOf(t, got[0]) != "a2" || got[0].Sequence != 2 {
		t.Errorf("got = %+v", got)
	}
}

func TestDeliverable_RespectsLimit(t *testing.T) {
	f := newFixture(t)
	f.append(t, message(uuid.New().String(), "a"),
		message(uuid.New().String(), "b"))
	got, err := f.store.Deliverable(t.Context(), 1)
	if err != nil || len(got) != 1 {
		t.Fatalf("got = %v, err = %v", got, err)
	}
}

// recorder は配られたイベントの名前を順に記録する。
type recorder struct {
	names []string
	fail  map[string]error
}

func (r *recorder) handle(
	_ context.Context, _ string, e event,
) error {
	if err, ok := r.fail[e.Name]; ok {
		return err
	}
	r.names = append(r.names, e.Name)
	return nil
}

func (f fixture) relay(
	r *recorder, buf *bytes.Buffer, opts outbox.RelayOptions,
) *outbox.Relay {
	logger := slog.New(slog.NewJSONHandler(buf, nil))
	relay := outbox.NewRelay(f.store, logger, opts)
	relay.Subscribe(eventType, "test", outbox.On(r.handle))
	return relay
}

func drain(t *testing.T, relay *outbox.Relay) {
	t.Helper()
	for range 100 {
		n, err := relay.RunOnce(t.Context())
		if err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
		if n == 0 {
			return
		}
	}
	t.Fatal("配信が終わらない")
}

func TestRelay_DeliversInOrderWithinAggregate(t *testing.T) {
	f := newFixture(t)
	a := uuid.New().String()
	f.append(t, message(a, "1"), message(a, "2"), message(a, "3"))
	var r recorder
	var buf bytes.Buffer
	drain(t, f.relay(&r, &buf, outbox.RelayOptions{}))
	want := []string{"1", "2", "3"}
	if diff := cmp.Diff(want, r.names); diff != "" {
		t.Errorf("順序 (-want +got):\n%s", diff)
	}
}

func TestRelay_FailedEventBlocksSameAggregateOnly(t *testing.T) {
	f := newFixture(t)
	a, b := uuid.New().String(), uuid.New().String()
	f.append(t, message(a, "a1"), message(a, "a2"),
		message(b, "b1"), message(b, "b2"))
	r := recorder{fail: map[string]error{
		"a1": errors.New("boom"),
	}}
	var buf bytes.Buffer
	relay := f.relay(&r, &buf, outbox.RelayOptions{})
	drain(t, relay)

	want := []string{"b1", "b2"}
	if diff := cmp.Diff(want, r.names); diff != "" {
		t.Fatalf("失敗中 (-want +got):\n%s", diff)
	}
	var attempts int
	var lastError string
	err := f.pool.QueryRow(t.Context(),
		`SELECT attempts, last_error FROM outbox
		 WHERE aggregate_id = $1 AND sequence = 1`, a).
		Scan(&attempts, &lastError)
	if err != nil {
		t.Fatalf("QueryRow: %v", err)
	}
	if attempts < 1 || lastError != "test: boom" {
		t.Errorf("attempts = %d, last_error = %q",
			attempts, lastError)
	}

	// 原因が直れば、止まっていた順に配信される
	r.fail = nil
	drain(t, relay)
	want = []string{"b1", "b2", "a1", "a2"}
	if diff := cmp.Diff(want, r.names); diff != "" {
		t.Errorf("回復後 (-want +got):\n%s", diff)
	}
}

// 失敗し続ける集約が1回の配信の枠を埋めても、ほかの集約は配られる。
// 失敗した集約の後のイベントは追い越さない。
func TestRelay_FailuresDoNotFillTheBatch(t *testing.T) {
	f := newFixture(t)
	failing := uuid.New().String()
	f.append(t, message(failing, "stuck1"),
		message(failing, "stuck2"))
	r := recorder{fail: map[string]error{
		"stuck1": errors.New("boom"),
	}}
	const batch = 100
	for i := range batch - 1 {
		name := fmt.Sprintf("failing%d", i)
		r.fail[name] = errors.New("boom")
		f.append(t, message(uuid.New().String(), name))
	}
	healthy := uuid.New().String()
	f.append(t, message(healthy, "ok1"), message(healthy, "ok2"))
	var buf bytes.Buffer
	relay := f.relay(&r, &buf, outbox.RelayOptions{BatchSize: batch})

	for range 3 {
		if _, err := relay.RunOnce(t.Context()); err != nil {
			t.Fatalf("RunOnce: %v", err)
		}
	}
	want := []string{"ok1", "ok2"}
	if diff := cmp.Diff(want, r.names); diff != "" {
		t.Fatalf("失敗中 (-want +got):\n%s", diff)
	}

	// 原因が直れば、止まっていた集約も順に配られる
	r.fail, r.names = nil, nil
	drain(t, relay)
	stuck1 := slices.Index(r.names, "stuck1")
	stuck2 := slices.Index(r.names, "stuck2")
	if len(r.names) != batch+1 || stuck1 < 0 || stuck1 > stuck2 {
		t.Errorf("回復後 = %v", r.names)
	}
}

// markFailsOnce は最初の MarkPublished を失敗させる。受け手の処理が
// 終わり、配信済みの記録が残らなかった状況を作る。
type markFailsOnce struct {
	outbox.RelayStore
	failed bool
}

func (s *markFailsOnce) MarkPublished(
	ctx context.Context, id int64,
) error {
	if !s.failed {
		s.failed = true
		return errors.New("mark")
	}
	return s.RelayStore.MarkPublished(ctx, id)
}

func TestRelay_RedeliversWhenMarkFails(t *testing.T) {
	f := newFixture(t)
	f.append(t, message(uuid.New().String(), "a"))
	var r recorder
	var buf bytes.Buffer
	relay := outbox.NewRelay(
		&markFailsOnce{RelayStore: f.store},
		slog.New(slog.NewJSONHandler(&buf, nil)),
		outbox.RelayOptions{})
	relay.Subscribe(eventType, "test", outbox.On(r.handle))

	n, err := relay.RunOnce(t.Context())
	if n != 0 || err == nil {
		t.Fatalf("1回目: n = %d, err = %v", n, err)
	}
	n, err = relay.RunOnce(t.Context())
	if n != 1 || err != nil {
		t.Fatalf("2回目: n = %d, err = %v", n, err)
	}
	want := []string{"a", "a"}
	if diff := cmp.Diff(want, r.names); diff != "" {
		t.Errorf("配信 (-want +got):\n%s", diff)
	}
}

func TestRelay_DeliversToEverySubscriber(t *testing.T) {
	f := newFixture(t)
	f.append(t, message(uuid.New().String(), "a"))
	var first, second recorder
	second.fail = map[string]error{"a": errors.New("boom")}
	var buf bytes.Buffer
	relay := f.relay(&first, &buf, outbox.RelayOptions{})
	relay.Subscribe(eventType, "second", outbox.On(second.handle))

	n, err := relay.RunOnce(t.Context())
	if n != 0 || err != nil {
		t.Fatalf("失敗中: n = %d, err = %v", n, err)
	}
	// 1人でも失敗すれば配信済みにしない。2回受ける購読者がいるので、
	// 受け手の側で処理済みを判定する
	second.fail = nil
	drain(t, relay)
	if len(first.names) != 2 || len(second.names) != 1 {
		t.Errorf("first = %v, second = %v",
			first.names, second.names)
	}
}

func TestRelay_NoSubscriberIsPublished(t *testing.T) {
	f := newFixture(t)
	m := message(uuid.New().String(), "a")
	m.EventType = "nobody.listens"
	f.append(t, m)
	var r recorder
	var buf bytes.Buffer
	relay := f.relay(&r, &buf, outbox.RelayOptions{})
	n, err := relay.RunOnce(t.Context())
	if n != 1 || err != nil {
		t.Fatalf("n = %d, err = %v", n, err)
	}
	got, err := f.store.Deliverable(t.Context(), 10)
	if err != nil || len(got) != 0 {
		t.Fatalf("got = %v, err = %v", got, err)
	}
}

// 許容時間の前は警告、超えたらエラーの水準で記録して再配信を続ける。
func TestRelay_LogsStalledEventAsError(t *testing.T) {
	cases := []struct {
		name  string
		after time.Duration
		level string
	}{
		{"許容時間の内", time.Hour, "WARN"},
		{"許容時間の外", time.Nanosecond, "ERROR"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			f := newFixture(t)
			a := uuid.New().String()
			f.append(t, message(a, "a"))
			r := recorder{fail: map[string]error{
				"a": errors.New("boom"),
			}}
			var buf bytes.Buffer
			relay := f.relay(&r, &buf, outbox.RelayOptions{
				StallAfter: c.after,
			})
			for range 2 {
				if _, err := relay.RunOnce(t.Context()); err != nil {
					t.Fatalf("RunOnce: %v", err)
				}
			}
			got := buf.String()
			if strings.Count(got, `"level":"`+c.level+`"`) != 2 {
				t.Errorf("log = %s", got)
			}
			if !strings.Contains(got, a) ||
				!strings.Contains(got, "boom") {
				t.Errorf("log = %s", got)
			}
			// 止めずに再配信を続ける
			head, err := f.store.Deliverable(t.Context(), 10)
			if err != nil || len(head) != 1 ||
				head[0].Attempts != 2 {
				t.Errorf("head = %+v, err = %v", head, err)
			}
		})
	}
}

// 停止の途中の失敗は原因の記録に数えず、次の起動で配信する。
func TestRelay_StopsWithoutRecordingCanceledDelivery(t *testing.T) {
	f := newFixture(t)
	f.append(t, message(uuid.New().String(), "a"))
	ctx, cancel := context.WithCancel(t.Context())
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	relay := outbox.NewRelay(f.store, logger, outbox.RelayOptions{})
	relay.Subscribe(eventType, "test", func(
		ctx context.Context, _ outbox.Delivery,
	) error {
		cancel()
		return ctx.Err()
	})
	_, err := relay.RunOnce(ctx)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
	head, err := f.store.Deliverable(t.Context(), 10)
	if err != nil || len(head) != 1 || head[0].Attempts != 0 {
		t.Fatalf("head = %+v, err = %v", head, err)
	}
}

func TestRun_DeliversUntilCanceled(t *testing.T) {
	f := newFixture(t)
	a := uuid.New().String()
	f.append(t, message(a, "1"), message(a, "2"))
	ctx, cancel := context.WithCancel(t.Context())
	var names []string
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	relay := outbox.NewRelay(f.store, logger,
		outbox.RelayOptions{Interval: time.Millisecond})
	relay.Subscribe(eventType, "test", outbox.On(func(
		_ context.Context, _ string, e event,
	) error {
		names = append(names, e.Name)
		if e.Name == "2" {
			cancel()
		}
		return nil
	}))
	done := make(chan error, 1)
	go func() { done <- relay.Run(ctx) }()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("err = %v", err)
		}
	case <-time.After(30 * time.Second):
		t.Fatal("Run が止まらない")
	}
	if diff := cmp.Diff([]string{"1", "2"}, names); diff != "" {
		t.Errorf("順序 (-want +got):\n%s", diff)
	}
}
