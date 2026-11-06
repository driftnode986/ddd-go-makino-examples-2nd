package application_test

import (
	"testing"

	"example.com/trainingbooking/internal/booking/application/querytest"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

func TestBookingQueries_Memory(t *testing.T) {
	querytest.Run(t, func(*testing.T) querytest.Fixture {
		repo := memory.NewBookingRepository()
		return querytest.Fixture{
			Repo:    repo,
			Tx:      memtx.NewTransactor(),
			Queries: memory.NewBookingQueries(repo),
		}
	})
}
