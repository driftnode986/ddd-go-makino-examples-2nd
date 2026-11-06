//go:build integration

// Package dbtest は統合テスト用の PostgreSQL を提供する。
package dbtest

import (
	"context"
	"fmt"
	"io/fs"
	"slices"
	"strings"
	"sync"
	"testing"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

const image = "postgres:18-alpine"

// コンテナはパッケージのテスト全体で1つ起動する。
// 停止は testcontainers のリーパーに任せる。
var container = sync.OnceValues(startContainer)

// #region start

func startContainer() (*postgres.PostgresContainer, error) {
	return postgres.Run(context.Background(), image,
		postgres.WithDatabase("postgres"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.BasicWaitStrategies(),
	)
}

// #endregion

// #region new

// New は新しい空のデータベースを作り、migrations を流す。
// テストどうしはデータベースで分離される。
func New(t *testing.T, migrations ...fs.FS) *pgxpool.Pool {
	t.Helper()
	ctx := t.Context()
	c, err := container()
	if err != nil {
		t.Fatalf("コンテナの起動: %v", err)
	}
	admin, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("接続文字列: %v", err)
	}
	hex := strings.ReplaceAll(uuid.New().String(), "-", "")
	name := "test_" + hex
	createDatabase(t, admin, name)

	cfg, err := pgxpool.ParseConfig(admin)
	if err != nil {
		t.Fatalf("接続設定: %v", err)
	}
	cfg.ConnConfig.Database = name
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatalf("接続: %v", err)
	}
	t.Cleanup(pool.Close)
	for _, m := range migrations {
		migrate(t, pool, m)
	}
	return pool
}

// #endregion

func createDatabase(t *testing.T, admin, name string) {
	t.Helper()
	ctx := t.Context()
	conn, err := pgx.Connect(ctx, admin)
	if err != nil {
		t.Fatalf("管理用の接続: %v", err)
	}
	defer func() { _ = conn.Close(context.Background()) }()
	_, err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE %s", name))
	if err != nil {
		t.Fatalf("CREATE DATABASE: %v", err)
	}
}

// migrate はファイル名の順に SQL を流す。
func migrate(t *testing.T, pool *pgxpool.Pool, m fs.FS) {
	t.Helper()
	paths, err := fs.Glob(m, "*/*.sql")
	if err != nil {
		t.Fatalf("マイグレーションの列挙: %v", err)
	}
	if len(paths) == 0 {
		t.Fatal("マイグレーションが1件もない")
	}
	slices.Sort(paths)
	for _, p := range paths {
		sql, err := fs.ReadFile(m, p)
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
		_, err = pool.Exec(t.Context(), string(sql))
		if err != nil {
			t.Fatalf("%s: %v", p, err)
		}
	}
}
