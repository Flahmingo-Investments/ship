package ship

import (
	"context"
)

// Publisher publishes message to a given topic.
type Publisher interface {
	// Publish publishes provided messages to given topic.
	Publish(topic string, messages ...*Message) error
}

// Subscriber subscribes to a given subscription.
type Subscriber interface {
	// Subscribe subscribe to a given subscription.
	Subscribe(subscription string, handler MessageHandler) error
	SubscribeRaw(subscription string, handler RawMessageHandler) error
}

// PubSub groups both Publisher and Subscriber methods together.
type PubSub interface {
	Publisher
	Subscriber
}

// RawMessageHandler provides method to handle a received message.
type RawMessageHandler interface {
	// HandleRawMessage handles a received message from Subscriber.
	HandleRawMessage(ctx context.Context, m *RawMessage) error
}

// MessageHandler provides method to handle a received message.
type MessageHandler interface {
	// HandleMessage handles a received message from Subscriber.
	HandleMessage(ctx context.Context, m *Message) error
}

// MessageHandlerFunc type is an adapter to allow the use of ordinary functions
// as Message handlers. If f is a function with the appropriate signature,
// MessageHandlerFunc(f) is a  Handler that calls f.
type MessageHandlerFunc func(context.Context, *Message) error

// HandleMessage handles a received message from Subscriber.
func (f MessageHandlerFunc) HandleMessage(ctx context.Context, m *Message) error {
	return f(ctx, m)
}
