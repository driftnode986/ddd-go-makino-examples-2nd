package domain_test

import (
	"errors"
	"math"
	"testing"

	"example.com/trainingbooking/internal/booking/domain"
)

func yen(t *testing.T, amount int64) domain.Money {
	t.Helper()
	m, err := domain.NewMoney(amount)
	if err != nil {
		t.Fatalf("NewMoney(%d): %v", amount, err)
	}
	return m
}

func TestMoney_RejectsNegative(t *testing.T) {
	_, err := domain.NewMoney(-1)
	if !errors.Is(err, domain.ErrNegativeMoney) {
		t.Fatalf("NewMoney(-1) = %v", err)
	}
}

func TestMoney_Add(t *testing.T) {
	got, err := yen(t, 1000).Add(yen(t, 500))
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if got.Amount() != 1500 {
		t.Fatalf("Amount = %d", got.Amount())
	}
}

func TestMoney_Multiply(t *testing.T) {
	got, err := yen(t, 300).Multiply(3)
	if err != nil {
		t.Fatalf("Multiply: %v", err)
	}
	if !got.Equal(yen(t, 900)) {
		t.Fatalf("Amount = %d", got.Amount())
	}
	zero, err := yen(t, 300).Multiply(0)
	if err != nil || !zero.IsZero() {
		t.Fatalf("Multiply(0) = %v, %v", zero, err)
	}
	_, err = yen(t, 300).Multiply(-1)
	if !errors.Is(err, domain.ErrNegativeMoney) {
		t.Fatalf("Multiply(-1) = %v", err)
	}
}

func TestMoney_Overflow(t *testing.T) {
	limit := yen(t, math.MaxInt64)
	if _, err := limit.Add(yen(t, 1)); !errors.Is(
		err, domain.ErrMoneyOverflow) {
		t.Errorf("Add = %v", err)
	}
	if _, err := limit.Multiply(2); !errors.Is(
		err, domain.ErrMoneyOverflow) {
		t.Errorf("Multiply = %v", err)
	}
	// 上限ちょうどはあふれない
	half := yen(t, math.MaxInt64/2)
	got, err := half.Multiply(2)
	if err != nil {
		t.Fatalf("Multiply: %v", err)
	}
	if got.Amount() != math.MaxInt64-1 {
		t.Errorf("Amount = %d", got.Amount())
	}
	if _, err := limit.Add(yen(t, 0)); err != nil {
		t.Errorf("Add(0) = %v", err)
	}
}

func TestZeroValues_AreDetected(t *testing.T) {
	var m domain.Money
	if !m.IsZero() {
		t.Error("var で宣言した Money は IsZero が true")
	}
	if yen(t, 1).IsZero() {
		t.Error("1円は IsZero が false")
	}
	var s domain.TimeSlot
	if !s.IsZero() {
		t.Error("var で宣言した TimeSlot は IsZero が true")
	}
}
