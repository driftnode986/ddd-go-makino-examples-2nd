// Package postgres は Identity の PostgreSQL 実装である。
package postgres

import (
	"context"
	"embed"
	"errors"
	"fmt"

	"example.com/trainingbooking/internal/identity/domain"
	"example.com/trainingbooking/internal/identity/infrastructure/postgres/sqlcgen"
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

// MemberRepository は Member を PostgreSQL に保存する。
type MemberRepository struct {
	pool *pgxpool.Pool
}

// NewMemberRepository は MemberRepository を返す。
func NewMemberRepository(pool *pgxpool.Pool) *MemberRepository {
	return &MemberRepository{pool: pool}
}

// Add は新しい会員を ctx の Tx の中で保存する。
func (r *MemberRepository) Add(
	ctx context.Context, m *domain.Member,
) error {
	tx, ok := pgtx.From(ctx)
	if !ok {
		return txn.ErrNoTransaction
	}
	s := m.State()
	err := sqlcgen.New(tx).InsertMember(ctx,
		sqlcgen.InsertMemberParams{
			ID:    s.ID.String(),
			Email: s.Email,
			Name:  s.Name,
		})
	pgErr, ok := errors.AsType[*pgconn.PgError](err)
	if ok && pgErr.Code == codeUniqueViolation {
		return fmt.Errorf("%w: %w", domain.ErrDuplicate, err)
	}
	return err
}

// FindByID は保存された中身から集約を組み立てる。
func (r *MemberRepository) FindByID(
	ctx context.Context, id ids.MemberID,
) (*domain.Member, error) {
	row, err := sqlcgen.New(r.pool).FindMember(ctx, id.String())
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %s", domain.ErrNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	return domain.ReconstructMember(domain.MemberState{
		ID:    id,
		Email: row.Email,
		Name:  row.Name,
	})
}
