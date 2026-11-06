package domain_test

import (
	"errors"
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"github.com/google/go-cmp/cmp"
)

func validState(t *testing.T) domain.BookingState {
	t.Helper()
	b := newBooking(t)
	if err := b.AddOption(
		option(t, "シューズ", 500, 2), earlier); err != nil {
		t.Fatalf("AddOption: %v", err)
	}
	if err := b.ApplyDiscount(yen(t, 100)); err != nil {
		t.Fatalf("ApplyDiscount: %v", err)
	}
	return b.State()
}

// #region round-trip

func TestReconstruct_RoundTrip(t *testing.T) {
	want := validState(t)
	b, err := domain.ReconstructBooking(want)
	if err != nil {
		t.Fatalf("ReconstructBooking: %v", err)
	}
	if diff := cmp.Diff(want, b.State()); diff != "" {
		t.Fatalf("差分 (-want +got):\n%s", diff)
	}
	if !b.Total().Equal(yen(t, 3900)) {
		t.Fatalf("Total = %d", b.Total().Amount())
	}
	got := b.Slot()
	if !got.Equal(slot(t, lessonStart, time.Hour)) {
		t.Fatalf("Slot = %v", got)
	}
}

// #endregion

func TestReconstruct_RejectsDiscountOverSubtotal(t *testing.T) {
	st := validState(t)
	st.DiscountYen = 4001 // 料金3000と明細1000の合計を超える
	_, err := domain.ReconstructBooking(st)
	if !errors.Is(err, domain.ErrInvalidBooking) ||
		!errors.Is(err, domain.ErrDiscountTooLarge) {
		t.Fatalf("err = %v", err)
	}
}

func TestReconstruct_RejectsUnknownStatus(t *testing.T) {
	st := validState(t)
	st.Status = domain.Status("unknown")
	_, err := domain.ReconstructBooking(st)
	if !errors.Is(err, domain.ErrInvalidBooking) {
		t.Fatalf("err = %v", err)
	}
}

// #region allows-past

func TestReconstruct_AllowsPastStart(t *testing.T) {
	st := validState(t)
	// 開始が過去でも、営業時間外になった予約でも読み戻せる
	st.Start = time.Date(2001, time.January, 1, 3, 15, 0, 0, jst)
	st.End = st.Start.Add(time.Hour)
	if _, err := domain.ReconstructBooking(st); err != nil {
		t.Fatalf("err = %v", err)
	}
}

// #endregion

func TestReconstruct_RecordsNoEvent(t *testing.T) {
	b, err := domain.ReconstructBooking(validState(t))
	if err != nil {
		t.Fatalf("ReconstructBooking: %v", err)
	}
	if got := b.PullEvents(); len(got) != 0 {
		t.Fatalf("events = %v", got)
	}
}

func TestReconstruct_RejectsBrokenState(t *testing.T) {
	sixOptions := make([]domain.OptionState, 6)
	for i := range sixOptions {
		sixOptions[i] = domain.OptionState{
			Name:         string(rune('a' + i)),
			UnitPriceYen: 1,
			Quantity:     1,
		}
	}
	cases := []struct {
		name   string
		mutate func(*domain.BookingState)
	}{
		{"予約 ID がゼロ", func(s *domain.BookingState) {
			s.ID = ids.BookingID{}
		}},
		{"会員 ID がゼロ", func(s *domain.BookingState) {
			s.MemberID = ids.MemberID{}
		}},
		{"レッスン ID がゼロ", func(s *domain.BookingState) {
			s.LessonID = ids.LessonID{}
		}},
		{"状態が空", func(s *domain.BookingState) {
			s.Status = ""
		}},
		{"開始がゼロ", func(s *domain.BookingState) {
			s.Start = time.Time{}
		}},
		{"終了が開始と同じ", func(s *domain.BookingState) {
			s.End = s.Start
		}},
		{"料金が負", func(s *domain.BookingState) {
			s.FeeYen = -1
		}},
		{"料金が0", func(s *domain.BookingState) {
			s.FeeYen = 0
		}},
		{"割引が負", func(s *domain.BookingState) {
			s.DiscountYen = -1
		}},
		{"割引が小計を超える", func(s *domain.BookingState) {
			s.DiscountYen = 1_000_000
		}},
		{"版が負", func(s *domain.BookingState) {
			s.Version = -1
		}},
		{"明細の単価が負", func(s *domain.BookingState) {
			s.Options[0].UnitPriceYen = -1
		}},
		{"明細の数量が0", func(s *domain.BookingState) {
			s.Options[0].Quantity = 0
		}},
		{"明細の名前が空", func(s *domain.BookingState) {
			s.Options[0].Name = ""
		}},
		{"明細の名前が重複", func(s *domain.BookingState) {
			s.Options = append(s.Options, s.Options[0])
		}},
		{"明細が上限を超える", func(s *domain.BookingState) {
			s.Options = sixOptions
			s.DiscountYen = 0
		}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := validState(t)
			c.mutate(&st)
			b, err := domain.ReconstructBooking(st)
			if !errors.Is(err, domain.ErrInvalidBooking) {
				t.Fatalf("err = %v", err)
			}
			if b != nil {
				t.Fatal("エラーのときは nil を返す")
			}
		})
	}
}

func TestReconstruct_RejectsZeroValueBooking(t *testing.T) {
	var b domain.Booking
	_, err := domain.ReconstructBooking(b.State())
	if !errors.Is(err, domain.ErrInvalidBooking) {
		t.Fatalf("err = %v", err)
	}
}

// 保存された時間帯は、新規作成のときだけの条件を問わず読み戻せる。
func TestReconstructTimeSlot_SkipsCreationRules(t *testing.T) {
	start := at(1, 8) // 営業時間の外
	end := start.Add(30 * time.Minute)
	s, err := domain.ReconstructTimeSlot(start, end)
	if err != nil {
		t.Fatalf("ReconstructTimeSlot: %v", err)
	}
	if !s.Start().Equal(start) || !s.End().Equal(end) {
		t.Errorf("slot = %v to %v", s.Start(), s.End())
	}
}

func TestReconstructTimeSlot_RejectsBrokenValues(t *testing.T) {
	start := at(1, 10)
	cases := []struct {
		name       string
		start, end time.Time
	}{
		{"開始が無い", time.Time{}, start},
		{"終了が開始と同じ", start, start},
		{"終了が開始より前", start, start.Add(-time.Hour)},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := domain.ReconstructTimeSlot(c.start, c.end)
			if !errors.Is(err, domain.ErrInvalidTimeSlot) {
				t.Fatalf("err = %v", err)
			}
		})
	}
}
