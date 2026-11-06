package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/infrastructure/postgres/sqlcgen"
	"example.com/trainingbooking/internal/ids"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations はスキーマの SQL である。
//
//go:embed migrations/*.sql
var Migrations embed.FS

// PostgreSQL のエラーコードである。
const (
	codeUniqueViolation    = "23505"
	codeExclusionViolation = "23P01"
)

// BookingRepository は Booking を PostgreSQL に保存する。
type BookingRepository struct {
	pool *pgxpool.Pool
}

// NewBookingRepository は BookingRepository を返す。
func NewBookingRepository(pool *pgxpool.Pool) *BookingRepository {
	return &BookingRepository{pool: pool}
}

// Add は親と明細を ctx の Tx の中で保存する。
//
// 重なる予約を同時に書くと、排他制約の検査どうしが互いの Tx を
// 待ってデッドロック（40P01）になることがある。会員ごとの
// アドバイザリロックで順番に並べ、必ず 23P01 で判定させる。
func (r *BookingRepository) Add(
	ctx context.Context, b *domain.Booking,
) error {
	s, q, err := prepare(ctx, b)
	if err != nil {
		return err
	}
	return translate(insertBooking(ctx, q, s))
}

// Save は読み込んだときの版で判定し、ctx の Tx の中で更新する。
// b の版は変えない。状態の更新も排他制約の検査を走らせるので、
// Add と同じロックを取る。
func (r *BookingRepository) Save(
	ctx context.Context, b *domain.Booking,
) error {
	s, q, err := prepare(ctx, b)
	if err != nil {
		return err
	}
	return translate(updateBooking(ctx, q, s))
}

func insertBooking(
	ctx context.Context, q *sqlcgen.Queries, s domain.BookingState,
) error {
	err := q.LockMember(ctx, s.MemberID.String())
	if err != nil {
		return err
	}
	err = q.InsertBooking(ctx, sqlcgen.InsertBookingParams{
		ID:          s.ID.String(),
		MemberID:    s.MemberID.String(),
		LessonID:    s.LessonID.String(),
		StartsAt:    s.Start,
		EndsAt:      s.End,
		FeeYen:      s.FeeYen,
		DiscountYen: s.DiscountYen,
		Status:      string(s.Status),
		Version:     int64(s.Version),
	})
	if err != nil {
		return err
	}
	return insertOptions(ctx, q, s)
}

func updateBooking(
	ctx context.Context, q *sqlcgen.Queries, s domain.BookingState,
) error {
	err := q.LockMember(ctx, s.MemberID.String())
	if err != nil {
		return err
	}
	params := sqlcgen.UpdateBookingParams{
		ID:          s.ID.String(),
		Version:     int64(s.Version),
		Status:      string(s.Status),
		DiscountYen: s.DiscountYen,
	}
	rows, err := q.UpdateBooking(ctx, params)
	if err != nil {
		return err
	}
	if rows == 0 {
		return missingOrConflict(ctx, q, s.ID)
	}
	err = q.DeleteBookingOptions(ctx, s.ID.String())
	if err != nil {
		return err
	}
	return insertOptions(ctx, q, s)
}

// FindByID は親と明細を1つの SQL で読み、集約を組み立てる。
func (r *BookingRepository) FindByID(
	ctx context.Context, id ids.BookingID,
) (*domain.Booking, error) {
	rows, err := r.queries(ctx).FindBookingWithOptions(
		ctx, id.String())
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	s, err := toState(id, rows)
	if err != nil {
		return nil, err
	}
	return domain.ReconstructBooking(s)
}

// queries は ctx に Tx があれば Tx で、無ければプールで実行する。
func (r *BookingRepository) queries(
	ctx context.Context,
) *sqlcgen.Queries {
	if tx, ok := txFrom(ctx); ok {
		return sqlcgen.New(tx)
	}
	return sqlcgen.New(r.pool)
}

// prepare は保存できる集約かと ctx の Tx を確かめて、
// 中身のコピーと Tx に結び付いたクエリを返す。
func prepare(
	ctx context.Context, b *domain.Booking,
) (domain.BookingState, *sqlcgen.Queries, error) {
	if err := ctx.Err(); err != nil {
		return domain.BookingState{}, nil, err
	}
	s := b.State()
	if _, err := domain.ReconstructBooking(s); err != nil {
		return domain.BookingState{}, nil, err
	}
	tx, ok := txFrom(ctx)
	if !ok {
		return domain.BookingState{}, nil, domain.ErrNoTransaction
	}
	return s, sqlcgen.New(tx), nil
}

func insertOptions(
	ctx context.Context, q *sqlcgen.Queries, s domain.BookingState,
) error {
	for i, o := range s.Options {
		err := q.InsertBookingOption(ctx,
			sqlcgen.InsertBookingOptionParams{
				BookingID:    s.ID.String(),
				Position:     int32(i),
				Name:         o.Name,
				UnitPriceYen: o.UnitPriceYen,
				Quantity:     int64(o.Quantity),
			})
		if err != nil {
			return err
		}
	}
	return nil
}

// missingOrConflict は更新が0行だった理由を判定する。
func missingOrConflict(
	ctx context.Context, q *sqlcgen.Queries, id ids.BookingID,
) error {
	exists, err := q.BookingExists(ctx, id.String())
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	return fmt.Errorf("%w: %s", domain.ErrVersionConflict, id)
}

func toState(
	id ids.BookingID, rows []sqlcgen.FindBookingWithOptionsRow,
) (domain.BookingState, error) {
	first := rows[0]
	member, err := ids.ParseID[ids.MemberMarker](
		first.MemberID)
	if err != nil {
		return domain.BookingState{}, err
	}
	lesson, err := ids.ParseID[ids.LessonMarker](
		first.LessonID)
	if err != nil {
		return domain.BookingState{}, err
	}
	s := domain.BookingState{
		ID:          id,
		MemberID:    member,
		LessonID:    lesson,
		Start:       first.StartsAt,
		End:         first.EndsAt,
		FeeYen:      first.FeeYen,
		DiscountYen: first.DiscountYen,
		Status:      domain.Status(first.Status),
		Version:     int(first.Version),
		Options:     make([]domain.OptionState, 0, len(rows)),
	}
	for _, row := range rows {
		// 明細が無い予約は、明細の列が NULL の1行になる
		if row.OptionName == nil {
			continue
		}
		s.Options = append(s.Options, domain.OptionState{
			Name:         *row.OptionName,
			UnitPriceYen: *row.OptionUnitPriceYen,
			Quantity:     int(*row.OptionQuantity),
		})
	}
	return s, nil
}

// translate は PostgreSQL のエラーをドメインのエラーに変換する。
func translate(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if !ok {
		return err
	}
	switch pgErr.Code {
	case codeUniqueViolation:
		return fmt.Errorf("%w: %w", domain.ErrDuplicate, err)
	case codeExclusionViolation:
		return fmt.Errorf("%w: %w", domain.ErrOverlap, err)
	}
	return err
}
