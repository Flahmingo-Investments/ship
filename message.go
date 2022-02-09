package ship

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// Metadata is an alias for map[string]string
type Metadata map[string]string

// Value implements the driver Valuer interface.
func (m Metadata) Value() (driver.Value, error) {
	return json.Marshal(m)
}

// Scan implements the sql Scanner interface.
func (m Metadata) Scan(v interface{}) error {
	if v == nil {
		return nil
	}
	switch data := v.(type) {
	case string:
		return json.Unmarshal([]byte(data), &m)
	case []byte:
		return json.Unmarshal(data, &m)
	default:
		return fmt.Errorf("cannot scan type %t into ship.Metadata", v)
	}
}

// Message is the fundamental unit for passing messages and events inside a
// ship application.
type Message struct {
	// ID is event id.
	ID string `db:"id" json:"id"`

	// Metadata about the event and message.
	Metadata Metadata `db:"metadata" json:"metadata"`

	// Type is event name.
	//
	// For example: UserOnBoarded, AccountClosed, etc.
	Type string `db:"type" json:"type"`

	// AggregateID is domain aggregate id for which we are storing the events for.
	AggregateID string `db:"aggregate_id" json:"aggregate_id"`

	// AggregateType is aggregate name.
	//
	// For example: User, Account, etc.
	AggregateType string `db:"aggregate_type" json:"aggregate_type"`

	// Data is event data that we are storing.
	Data Event `db:"data" json:"data"`

	// At, what time the event occurred.
	At time.Time `db:"at" json:"at"`

	// Version of the event or message.
	Version uint64 `db:"version" json:"version"`
}
