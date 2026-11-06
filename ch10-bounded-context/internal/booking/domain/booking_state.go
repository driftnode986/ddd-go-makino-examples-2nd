package domain

import (
	"fmt"
	"time"

	"example.com/trainingbooking/internal/ids"
)

// BookingState は保存と再構築のための集約の中身である。
// 金額は検証前の値も表せるように整数で持つ。
type BookingState struct {
	ID          ids.BookingID
	MemberID    ids.MemberID
	LessonID    ids.LessonID
	Start       time.Time
	End         time.Time
	FeeYen      int64
	DiscountYen int64
	Status      Status
	Version     int
	Options     []OptionState
}

// OptionState は明細の中身である。
type OptionState struct {
	Name         string
	UnitPriceYen int64
	Quantity     int
}

// State は集約の中身のコピーを返す。
func (b *Booking) State() BookingState {
	options := make([]OptionState, len(b.options))
	for i, o := range b.options {
		options[i] = OptionState{
			Name:         o.name,
			UnitPriceYen: o.unitPrice.yen,
			Quantity:     o.quantity,
		}
	}
	return BookingState{
		ID:          b.id,
		MemberID:    b.memberID,
		LessonID:    b.lessonID,
		Start:       b.slot.Start(),
		End:         b.slot.End(),
		FeeYen:      b.fee.yen,
		DiscountYen: b.discount.yen,
		Status:      b.status,
		Version:     b.version,
		Options:     options,
	}
}

// ReconstructBooking は保存された中身から集約を組み立てる。
// 開始が未来か、営業時間内かといった新規作成のときだけの条件は省く。
// 保存されていてはいけない値はエラーにする。イベントは記録しない。
func ReconstructBooking(s BookingState) (*Booking, error) {
	b, err := rebuild(s)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidBooking, err)
	}
	return b, nil
}

// ReconstructTimeSlot は保存された開始と終了から時間帯を組み立てる。
// 営業時間のような新規作成のときだけの条件は省く。
func ReconstructTimeSlot(start, end time.Time) (TimeSlot, error) {
	// Sub は Duration に入らない差を切り詰めるので、戻らない差も拒否する。
	d := end.Sub(start)
	if start.IsZero() || !end.After(start) ||
		!start.Add(d).Equal(end) {
		return TimeSlot{}, fmt.Errorf(
			"%w: %s to %s", ErrInvalidTimeSlot, start, end)
	}
	return TimeSlot{start: start, duration: d}, nil
}

func rebuild(s BookingState) (*Booking, error) {
	if s.ID.IsZero() || s.MemberID.IsZero() || s.LessonID.IsZero() {
		return nil, fmt.Errorf("id is missing")
	}
	if !s.Status.valid() {
		return nil, fmt.Errorf("unknown status %q", s.Status)
	}
	slot, err := ReconstructTimeSlot(s.Start, s.End)
	if err != nil {
		return nil, err
	}
	// DB の時刻は1マイクロ秒まで。それより細かい値は保存できない。
	if s.Start.Nanosecond()%1000 != 0 ||
		s.End.Nanosecond()%1000 != 0 {
		return nil, fmt.Errorf("slot %s to %s", s.Start, s.End)
	}
	if s.Version < 0 {
		return nil, fmt.Errorf("version %d", s.Version)
	}
	fee, err := NewMoney(s.FeeYen)
	if err != nil {
		return nil, err
	}
	if fee.IsZero() {
		return nil, fmt.Errorf("fee is zero")
	}
	discount, err := NewMoney(s.DiscountYen)
	if err != nil {
		return nil, err
	}
	options, err := rebuildOptions(s.Options)
	if err != nil {
		return nil, err
	}
	b := &Booking{
		id:       s.ID,
		memberID: s.MemberID,
		lessonID: s.LessonID,
		slot:     slot,
		fee:      fee,
		discount: discount,
		status:   s.Status,
		version:  s.Version,
	}
	if err := b.setOptions(options); err != nil {
		return nil, err
	}
	return b, nil
}

func rebuildOptions(states []OptionState) ([]BookingOption, error) {
	options := make([]BookingOption, 0, len(states))
	for _, s := range states {
		price, err := NewMoney(s.UnitPriceYen)
		if err != nil {
			return nil, err
		}
		o, err := NewBookingOption(s.Name, price, s.Quantity)
		if err != nil {
			return nil, err
		}
		options = append(options, o)
	}
	return options, nil
}
