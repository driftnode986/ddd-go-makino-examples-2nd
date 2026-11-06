package memory_test

import (
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
	"example.com/trainingbooking/internal/booking/domain/repositorytest"
	"example.com/trainingbooking/internal/booking/infrastructure/memory"
)

func TestBookingRepository_Contract(t *testing.T) {
	repositorytest.Run(t,
		func(*testing.T) domain.BookingRepository {
			return memory.NewBookingRepository()
		})
}
