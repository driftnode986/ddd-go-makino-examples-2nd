//go:build integration

package outbox_test

import (
	"testing"

	"example.com/trainingbooking/internal/platform/dbtest"
	"example.com/trainingbooking/internal/platform/outbox"
	"example.com/trainingbooking/internal/platform/txn/pgtx"
)

func TestProcessedEvents(t *testing.T) {
	pool := dbtest.New(t, outbox.Migrations)
	testProcessedEvents(t,
		outbox.NewProcessedEvents(pool), pgtx.NewTransactor(pool))
}
