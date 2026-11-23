package entities

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Date is a calendar date. On the JSON wire it is "YYYY-MM-DD" (the natural
// format for goals, plan sessions and climbing log entries), and in SQL it
// maps to a DATE column. Internally it is a time.Time at midnight UTC.
type Date time.Time

const dateLayout = "2006-01-02"

// Time returns the underlying time value.
func (d Date) Time() time.Time {
	return time.Time(d)
}

// IsZero reports whether the date is the zero value.
func (d Date) IsZero() bool {
	return d.Time().IsZero()
}

// String formats the date as YYYY-MM-DD.
func (d Date) String() string {
	return d.Time().UTC().Format(dateLayout)
}

// AddDays returns the date shifted by the given number of days.
func (d Date) AddDays(days int) Date {
	return Date(d.Time().AddDate(0, 0, days))
}

// After reports whether d is strictly after other.
func (d Date) After(other Date) bool {
	return d.Time().After(other.Time())
}

// MarshalJSON encodes the date as "YYYY-MM-DD".
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte(`null`), nil
	}
	return []byte(`"` + d.String() + `"`), nil
}

// UnmarshalJSON parses a "YYYY-MM-DD" date string.
func (d *Date) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return fmt.Errorf("date must be a \"YYYY-MM-DD\" string: %w", err)
	}
	parsed, err := time.Parse(dateLayout, value)
	if err != nil {
		return fmt.Errorf("date must be YYYY-MM-DD, got %q", value)
	}
	*d = Date(parsed)
	return nil
}

// Scan implements sql.Scanner for DATE columns.
func (d *Date) Scan(value any) error {
	scanned, ok := value.(time.Time)
	if !ok {
		return fmt.Errorf("date: cannot scan %T into Date", value)
	}
	*d = Date(scanned)
	return nil
}

// Value implements driver.Valuer for DATE columns.
func (d Date) Value() (driver.Value, error) {
	return d.Time(), nil
}
