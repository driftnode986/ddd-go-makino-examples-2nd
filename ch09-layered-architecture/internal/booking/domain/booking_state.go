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

func rebuild(s BookingState) (*Booking, error) {
	if s.ID.IsZero() || s.MemberID.IsZero() || s.LessonID.IsZero() {
		return nil, fmt.Errorf("id is missing")
	}
	if !s.Status.valid() {
		return nil, fmt.Errorf("unknown status %q", s.Status)
	}
	if s.Start.IsZero() || !s.End.After(s.Start) {
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
		slot: TimeSlot{
			start:    s.Start,
			duration: s.End.Sub(s.Start),
		},
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
