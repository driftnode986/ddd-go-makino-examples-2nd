// Package postgres は Catalog の PostgreSQL 実装である。
package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"example.com/trainingbooking/internal/catalog/domain"
	"example.com/trainingbooking/internal/catalog/infrastructure/postgres/sqlcgen"
	"example.com/trainingbooking/internal/ids"
	"example.com/trainingbooking/internal/platform/txn"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrations はスキーマの SQL である。
//
//go:embed migrations/*.sql
var Migrations embed.FS

const codeUniqueViolation = "23505"

// LessonRepository は Lesson を PostgreSQL に保存する。
type LessonRepository struct {
	pool *pgxpool.Pool
}

// NewLessonRepository は LessonRepository を返す。
func NewLessonRepository(pool *pgxpool.Pool) *LessonRepository {
	return &LessonRepository{pool: pool}
}

// Add は親と明細を ctx の Tx の中で保存する。
func (r *LessonRepository) Add(
	ctx context.Context, l *domain.Lesson,
) error {
	s, q, err := prepare(ctx, l)
	if err != nil {
		return err
	}
	return translate(insertLesson(ctx, q, s))
}

// Save は読み込んだときの版で判定し、ctx の Tx の中で席を更新する。
// 同時に保存すると、後の UPDATE は先の Tx の終わりを待ってから
// 0行になる。l の版は変えない。
func (r *LessonRepository) Save(
	ctx context.Context, l *domain.Lesson,
) error {
	s, q, err := prepare(ctx, l)
	if err != nil {
		return err
	}
	rows, err := q.BumpLessonVersion(ctx,
		sqlcgen.BumpLessonVersionParams{
			ID:      s.ID.String(),
			Version: int64(s.Version),
		})
	if err != nil {
		return err
	}
	if rows == 0 {
		return missingOrConflict(ctx, q, s.ID)
	}
	err = q.DeleteLessonSeats(ctx, s.ID.String())
	if err != nil {
		return err
	}
	return insertSeats(ctx, q, s)
}

// FindByID は親と明細を1つの SQL で読み、集約を組み立てる。
func (r *LessonRepository) FindByID(
	ctx context.Context, id ids.LessonID,
) (*domain.Lesson, error) {
	row, err := queriesFor(ctx, r.pool).FindLesson(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	s, err := toState(id, row)
	if err != nil {
		return nil, err
	}
	return domain.ReconstructLesson(s)
}

func queriesFor(
	ctx context.Context, pool *pgxpool.Pool,
) *sqlcgen.Queries {
	if tx, ok := pgtx.From(ctx); ok {
		return sqlcgen.New(tx)
	}
	return sqlcgen.New(pool)
}

func prepare(
	ctx context.Context, l *domain.Lesson,
) (domain.LessonState, *sqlcgen.Queries, error) {
	if err := ctx.Err(); err != nil {
		return domain.LessonState{}, nil, err
	}
	s := l.State()
	if _, err := domain.ReconstructLesson(s); err != nil {
		return domain.LessonState{}, nil, err
	}
	tx, ok := pgtx.From(ctx)
	if !ok {
		return domain.LessonState{}, nil, txn.ErrNoTransaction
	}
	return s, sqlcgen.New(tx), nil
}

func insertLesson(
	ctx context.Context, q *sqlcgen.Queries, s domain.LessonState,
) error {
	err := q.InsertLesson(ctx, sqlcgen.InsertLessonParams{
		ID:       s.ID.String(),
		Title:    s.Title,
		StartsAt: s.Start,
		EndsAt:   s.Start.Add(s.Duration),
		FeeYen:   s.FeeYen,
		Capacity: int32(s.Capacity),
		Version:  int64(s.Version),
	})
	if err != nil {
		return err
	}
	for i, o := range s.Options {
		err := q.InsertLessonOption(ctx,
			sqlcgen.InsertLessonOptionParams{
				LessonID:     s.ID.String(),
				Position:     int32(i),
				Name:         o.Name,
				UnitPriceYen: o.UnitPriceYen,
			})
		if err != nil {
			return err
		}
	}
	return insertSeats(ctx, q, s)
}

func insertSeats(
	ctx context.Context, q *sqlcgen.Queries, s domain.LessonState,
) error {
	for i, b := range s.Seats {
		err := q.InsertLessonSeat(ctx,
			sqlcgen.InsertLessonSeatParams{
				LessonID:  s.ID.String(),
				Position:  int32(i),
				BookingID: b.String(),
			})
		if err != nil {
			return err
		}
	}
	return nil
}

func missingOrConflict(
	ctx context.Context, q *sqlcgen.Queries, id ids.LessonID,
) error {
	exists, err := q.LessonExists(ctx, id.String())
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	return fmt.Errorf("%w: %s", domain.ErrVersionConflict, id)
}

func toState(
	id ids.LessonID, row sqlcgen.FindLessonRow,
) (domain.LessonState, error) {
	if len(row.OptionNames) != len(row.OptionPrices) {
		return domain.LessonState{}, fmt.Errorf(
			"option columns differ: %s", id)
	}
	s := domain.LessonState{
		ID:       id,
		Title:    row.Title,
		Start:    row.StartsAt,
		Duration: row.EndsAt.Sub(row.StartsAt),
		FeeYen:   row.FeeYen,
		Capacity: int(row.Capacity),
		Version:  int(row.Version),
	}
	for i, name := range row.OptionNames {
		s.Options = append(s.Options, domain.LessonOption{
			Name:         name,
			UnitPriceYen: row.OptionPrices[i],
		})
	}
	for _, seat := range row.Seats {
		b, err := ids.ParseID[ids.BookingMarker](seat)
		if err != nil {
			return domain.LessonState{}, err
		}
		s.Seats = append(s.Seats, b)
	}
	return s, nil
}

func translate(err error) error {
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if ok && pgErr.Code == codeUniqueViolation {
		return fmt.Errorf("%w: %w", domain.ErrDuplicate, err)
	}
	return err
}
