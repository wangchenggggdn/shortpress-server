package types

import (
	"fmt"
	"math"
	"strconv"
)

// Money represents money for API usage with automatic dollar/cent conversion
// Default currency is USD, stores value in cents internally
type Money int64

// Dollars returns the money value in dollars as float64
func (m Money) Dollars() float64 {
	return float64(m) / 100
}

// Cents returns the money value in cents as int64
func (m Money) Cents() int64 {
	return int64(m)
}

// FromDollars creates Money from dollar amount
func FromDollars(dollars float64) Money {
	// Use math.Round to handle floating point precision issues
	// e.g., 69.99 * 100 = 6998.9999... -> Round to 6999
	return Money(math.Round(dollars * 100))
}

// FromCents creates Money from cents amount
func FromCents(cents int64) Money {
	return Money(cents)
}

// MarshalJSON implements json.Marshaler (returns dollars for API)
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(fmt.Sprintf("%.2f", m.Dollars())), nil
}

// UnmarshalJSON implements json.Unmarshaler (accepts dollars from API)
func (m *Money) UnmarshalJSON(data []byte) error {
	str := string(data)
	dollars, err := strconv.ParseFloat(str, 64)
	if err != nil {
		return err
	}
	*m = FromDollars(dollars)
	return nil
}

// String returns formatted dollar amount
func (m Money) String() string {
	return fmt.Sprintf("$%.4f", m.Dollars())
}

// Add adds another Money value
func (m Money) Add(other Money) Money {
	return m + other
}

// Sub subtracts another Money value
func (m Money) Sub(other Money) Money {
	return m - other
}

// IsZero checks if the money value is zero
func (m Money) IsZero() bool {
	return m == 0
}
