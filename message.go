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

// RawMessage is the pure message and does not do any conversion.
type RawMessage struct {
	// ID identifies this message. This ID is assigned by the server and is
	// populated for Messages obtained from a subscription.
	//
	// This field is read-only.
	ID string

	// Data is the actual data in the message.
	Data []byte

	// Attributes represents the key-value pairs the current message is
	// labelled with.
	Attributes map[string]string

	// PublishTime is the time at which the message was published. This is
	// populated by the server for Messages obtained from a subscription.
	//
	// This field is read-only.
	PublishTime time.Time

	// OrderingKey identifies related messages for which publish order should
	// be respected. If empty string is used, message will be sent unordered.
	OrderingKey string
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
