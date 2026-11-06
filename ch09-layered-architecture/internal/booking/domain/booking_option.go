package domain

import "fmt"

// BookingOption は予約の明細である。シューズのレンタルなどを表す。
// 同じ予約の中では名前で区別する。
type BookingOption struct {
	name      string
	unitPrice Money
	quantity  int
	subtotal  Money
}

// NewBookingOption は明細を作る。
// 小計が計算できない組み合わせは拒否する。
func NewBookingOption(
	name string, unitPrice Money, quantity int,
) (BookingOption, error) {
	if name == "" || quantity <= 0 {
		return BookingOption{}, fmt.Errorf(
			"%w: name=%q quantity=%d",
			ErrInvalidOption, name, quantity)
	}
	subtotal, err := unitPrice.Multiply(quantity)
	if err != nil {
		return BookingOption{}, err
	}
	return BookingOption{
		name:      name,
		unitPrice: unitPrice,
		quantity:  quantity,
		subtotal:  subtotal,
	}, nil
}

// Name は明細の名前を返す。
func (o BookingOption) Name() string { return o.name }

// Quantity は数量を返す。
func (o BookingOption) Quantity() int { return o.quantity }

// Subtotal は単価と数量の積を返す。
func (o BookingOption) Subtotal() Money { return o.subtotal }

// Equal は全項目が同じとき true を返す。
func (o BookingOption) Equal(other BookingOption) bool {
	return o.name == other.name &&
		o.unitPrice.Equal(other.unitPrice) &&
		o.quantity == other.quantity
}

// IsZero は NewBookingOption を通していないとき true を返す。
func (o BookingOption) IsZero() bool { return o.name == "" }
