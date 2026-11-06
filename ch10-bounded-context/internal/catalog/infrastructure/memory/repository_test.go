package memory_test

import (
	"testing"

	"example.com/trainingbooking/internal/catalog/domain/repositorytest"
	"example.com/trainingbooking/internal/catalog/infrastructure/memory"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

func TestLessonRepository_Contract(t *testing.T) {
	repositorytest.Run(t,
		func(*testing.T) repositorytest.Fixture {
			return repositorytest.Fixture{
				Repo: memory.NewLessonRepository(),
				Tx:   memtx.NewTransactor(),
			}
		})
}
