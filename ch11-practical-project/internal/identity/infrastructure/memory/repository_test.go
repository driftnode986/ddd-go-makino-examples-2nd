package memory_test

import (
	"testing"

	"example.com/trainingbooking/internal/identity/domain/repositorytest"
	"example.com/trainingbooking/internal/identity/infrastructure/memory"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

func TestMemberRepository_Contract(t *testing.T) {
	repositorytest.Run(t,
		func(*testing.T) repositorytest.Fixture {
			return repositorytest.Fixture{
				Repo: memory.NewMemberRepository(),
				Tx:   memtx.NewTransactor(),
			}
		})
}
