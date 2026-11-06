package memory_test

import (
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/domain/repositorytest"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
	"example.com/trainingbooking/internal/platform/txn/memtx"
)

// #region contract

func TestBookingRepository_Contract(t *testing.T) {
	repositorytest.Run(t,
		func(*testing.T) (
			domain.BookingRepository, repositorytest.Transactor,
		) {
			return memory.NewBookingRepository(),
				memtx.NewTransactor()
		})
}

// #endregion
