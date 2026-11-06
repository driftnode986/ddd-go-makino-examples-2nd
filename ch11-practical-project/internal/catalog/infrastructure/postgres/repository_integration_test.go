//go:build integration

package postgres_test

import (
	"testing"

	"example.com/trainingbooking/internal/catalog/domain/repositorytest"
	"example.com/trainingbooking/internal/catalog/infrastructure/postgres"
	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

func TestLessonRepository_Contract(t *testing.T) {
	repositorytest.Run(t,
		func(t *testing.T) repositorytest.Fixture {
			pool := dbtest.New(t, postgres.Migrations)
			return repositorytest.Fixture{
				Repo: postgres.NewLessonRepository(pool),
				Tx:   pgtx.NewTransactor(pool),
			}
		})
}
