package ship

// Message is the fundamental unit for passing messages and events inside a
// ship application.
type Message struct {
	MessageID string
	Metadata  map[string]string
	Data      Event
}

// Clone creates a copy of the message.
func (m *Message) Clone() *Message {
	msg := &Message{
		MessageID: m.MessageID,
		Data:      m.Data,
	}

	for k, v := range m.Metadata {
		msg.Metadata[k] = v
	}

	return msg
}
