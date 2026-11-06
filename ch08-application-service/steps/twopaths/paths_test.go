package twopaths_test

import (
	"testing"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/steps/twopaths"
)

type fixedClock struct{ now time.Time }

func (c fixedClock) Now() time.Time { return c.now }

func yen(t *testing.T, v int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(v)
	if err != nil {
		t.Fatalf("NewMoney: %v", err)
	}
	return m
}

func TestTwoPathsDiffer(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)
	start := time.Date(2030, time.June, 1, 10, 0, 0, 0, jst)
	clk := fixedClock{start.AddDate(0, -1, 0)}
	slot, err := domain.NewTimeSlot(start, time.Hour, jst)
	if err != nil {
		t.Fatalf("NewTimeSlot: %v", err)
	}
	newBooking := func() *domain.Booking {
		b, err := domain.NewBooking(
			ids.NewID[ids.BookingMarker](),
			ids.NewID[ids.MemberMarker](),
			ids.NewID[ids.LessonMarker](), slot, yen(t, 3000), clk)
		if err != nil {
			t.Fatalf("NewBooking: %v", err)
		}
		return b
	}
	web, csv := newBooking(), newBooking()
	prices := map[string]domain.Money{"シューズ": yen(t, 300)}
	err = twopaths.FromHTTP(web, "シューズ", 2, prices, clk)
	if err != nil {
		t.Fatalf("FromHTTP: %v", err)
	}
	row := twopaths.Row{
		Name: "シューズ", Quantity: 2, UnitPrice: yen(t, 250),
	}
	if err := twopaths.FromCSV(csv, row, clk); err != nil {
		t.Fatalf("FromCSV: %v", err)
	}
	t.Logf("HTTP の経路: 合計 %d 円", web.Total().Amount())
	t.Logf("バッチの経路: 合計 %d 円", csv.Total().Amount())
	if web.Total().Equal(csv.Total()) {
		t.Fatal("合計が同じになった")
	}
}
