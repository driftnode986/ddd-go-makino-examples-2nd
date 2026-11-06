package application

import (
	"context"
	"errors"
	"time"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/ids"
)

// LessonTerms は予約に使うレッスンの条件である。
type LessonTerms struct {
	Fee      domain.Money
	Location *time.Location
}

// PlaceBookingInput は予約の入力である。
type PlaceBookingInput struct {
	MemberID string
	LessonID string
	Start    time.Time
	Duration time.Duration
	// CouponCode は任意である。割引率は受け取らない。
	CouponCode string
}

// PlaceBookingOutput は予約の結果である。
type PlaceBookingOutput struct {
	BookingID string
	Status    string
}

// PlaceBooking は予約を受け付ける。
type PlaceBooking struct {
	repo    domain.BookingRepository
	queries BookingQueries
	outbox  OutboxWriter
	tx      Transactor
	clock   domain.Clock
	coupons Coupons
	terms   LessonTerms
}

// NewPlaceBooking は PlaceBooking を返す。
func NewPlaceBooking(
	repo domain.BookingRepository,
	queries BookingQueries,
	outbox OutboxWriter,
	tx Transactor,
	clock domain.Clock,
	coupons Coupons,
	terms LessonTerms,
) *PlaceBooking {
	return &PlaceBooking{
		repo, queries, outbox, tx, clock, coupons, terms,
	}
}

// Execute は予約を pending で保存し、同じ Tx で BookingPlaced を
// outbox に書く。席の確保は配信の先で行う。
func (u *PlaceBooking) Execute(
	ctx context.Context, in PlaceBookingInput,
) (PlaceBookingOutput, error) {
	b, err := u.newBooking(ctx, in)
	if err != nil {
		return PlaceBookingOutput{}, err
	}
	if err := u.checkOverlap(ctx, b); err != nil {
		return PlaceBookingOutput{}, err
	}
	err = u.tx.Do(ctx, func(ctx context.Context) error {
		if err := u.repo.Add(ctx, b); err != nil {
			return err
		}
		return u.outbox.Append(ctx, b.PullEvents()...)
	})
	if err != nil {
		return PlaceBookingOutput{}, translate(err)
	}
	return PlaceBookingOutput{
		BookingID: b.ID().String(),
		Status:    string(b.Status()),
	}, nil
}

// checkOverlap は同じ会員の有効な予約と時間帯が重なれば拒否する。
// ここを同時に通った予約は、リポジトリの排他制約が止める。
func (u *PlaceBooking) checkOverlap(
	ctx context.Context, b *domain.Booking,
) error {
	existing, err := u.queries.ActiveSlotsByMember(ctx, b.MemberID())
	if err != nil {
		return translate(err)
	}
	err = domain.DetectOverlap(b.Slot(), existing)
	if err != nil {
		return translate(err)
	}
	return nil
}

// newBooking は入力と LessonTerms から集約を作る。
// 入力の誤りは KindInvalid にする。
func (u *PlaceBooking) newBooking(
	ctx context.Context, in PlaceBookingInput,
) (*domain.Booking, error) {
	member, err := ids.ParseID[ids.MemberMarker](in.MemberID)
	if err != nil {
		return nil, invalid("会員 ID の形式が正しくありません", err)
	}
	lesson, err := ids.ParseID[ids.LessonMarker](in.LessonID)
	if err != nil {
		return nil, invalid("レッスン ID の形式が正しくありません", err)
	}
	slot, err := domain.NewTimeSlot(
		in.Start, in.Duration, u.terms.Location)
	if err != nil {
		return nil, invalid("予約できない時間帯です", err)
	}
	b, err := domain.NewBooking(
		ids.NewID[ids.BookingMarker](),
		member, lesson, slot, u.terms.Fee, u.clock)
	if errors.Is(err, domain.ErrStartInPast) {
		return nil, invalid("過去の時間帯は予約できません", err)
	}
	if err != nil {
		return nil, translate(err)
	}
	if err := u.applyCoupon(ctx, b, in.CouponCode); err != nil {
		return nil, err
	}
	return b, nil
}

// applyCoupon はコードの Coupon で割引を計算して適用する。
// コードが空なら何もしない。割引率は入力で受け取らない。
func (u *PlaceBooking) applyCoupon(
	ctx context.Context, b *domain.Booking, code string,
) error {
	if code == "" {
		return nil
	}
	coupon, err := u.coupons.Find(ctx, code)
	if errors.Is(err, ErrCouponNotFound) {
		return invalid("クーポンのコードが正しくありません", err)
	}
	if err != nil {
		return translate(err)
	}
	discount, err := domain.CalculateDiscount(b.Total(), coupon)
	if err != nil {
		return translate(err)
	}
	if err := b.ApplyDiscount(discount); err != nil {
		return translate(err)
	}
	return nil
}
