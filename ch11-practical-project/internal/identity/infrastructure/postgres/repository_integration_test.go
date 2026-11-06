//go:build integration

package postgres_test

import (
	"testing"

	"example.com/trainingbooking/internal/identity/domain/repositorytest"
	"example.com/trainingbooking/internal/identity/infrastructure/postgres"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

func TestMemberRepository_Contract(t *testing.T) {
	repositorytest.Run(t,
		func(t *testing.T) repositorytest.Fixture {
			pool := dbtest.New(t, postgres.Migrations)
			return repositorytest.Fixture{
				Repo: postgres.NewMemberRepository(pool),
				Tx:   pgtx.NewTransactor(pool),
			}
		})
}
