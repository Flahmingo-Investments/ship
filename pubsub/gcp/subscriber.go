package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"reflect"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/Flahmingo-Investments/ship"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Compile time check.
var _ ship.Subscriber = (*PubSub)(nil)

// _bufferPool is a pool of bytes.Buffers.
var _bufferPool = sync.Pool{
	New: func() interface{} {
		return &bytes.Buffer{}
	},
}

// subInit check for existence of subscription name and returns it.
// If subscription does not exists, it will throw error.
func (p *PubSub) subInit(subName string) (*pubsub.Subscription, error) {
	sub := p.client.Subscription(subName)

	p.logger.Info("checking if subscription exists", zap.String("name", subName))
	exists, err := sub.Exists(p.ctx)
	if err != nil {
		return nil, errors.Wrapf(
			err, "could not check if subscription '%s' exists", subName,
		)
	}

	if !exists {
		return nil, errors.Errorf("subscription %s does not exists", subName)
	}

	return sub, nil
}

// Subscribe subscribes a handler to a given subscription.
// It stops receiving message in case of, panics.
//
// It is a non-blocking call.
// NOTE: to stop the subscriptions. Call Stop method.
//
// Example:
//	pubsub, err := New("projectID")
//	if err != nil {
//		// do something with error
//		return
//	}
//
//	pubsub.Subscribe("some-subscription-name", handler)
//	pubsub.Subscribe("some-subscription-name2", handler2)
//	pubsub.Subscribe("some-subscription-name3", handler3)
//
func (p *PubSub) Subscribe(
	subscription string, handler ship.MessageHandler,
) error {
	sub, err := p.subInit(subscription)
	if err != nil {
		return errors.WithStack(err)
	}

	p.logger.Info(
		"starting listener for subscription", zap.String("subscription", subscription),
	)
	go p.handle(handler, sub)

	return nil
}

type debeziumMessage struct {
	Payload payload `json:"payload,omitempty"`
}

type payload struct {
	ID            string            `json:"id"`
	Type          string            `json:"type"`
	Metadata      map[string]string `json:"metadata"`
	AggregateID   string            `json:"aggregate_id"`
	AggregateType string            `json:"aggregate_type"`
	At            time.Time         `json:"at"`
	Version       uint64            `json:"version"`
	Data          string            `json:"data"`
	Table         string            `json:"__table"`
	LSN           uint64            `json:"__lsn"`
	Deleted       string            `json:"__deleted"`
}

// handle takes a message handler and a subscription.
// nolint:funlen
func (p *PubSub) handle(h ship.MessageHandler, sub *pubsub.Subscription) {
	p.wg.Add(1)
	defer p.wg.Done()

	t := reflect.TypeOf(h)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	hName := t.Name()

	p.logger.Debug(
		"subscription started",
		zap.String("subscription", sub.String()),
		zap.String("handlerName", hName),
	)

	// Creating a context with cancel so, we can cancel the subscription in case
	// of panic.
	ctx, cancel := context.WithCancel(p.ctx)

	err := sub.Receive(ctx, func(ctx context.Context, msg *pubsub.Message) {
		// We don't want an unexpected error in consumer to take down whole application.
		// We'll try to recover from the panic and remove the subscription from listening.
		//
		// This protects an application from going in a continuous crash loop in a
		// orchestrated environment.
		defer func() {
			if r := recover(); r != nil {
				p.logger.Error(
					"[BUG]: recovered from a panic in subscription."+" "+
						"Nacking the received message and removing the subscription from listening.",
					zap.String("subscription", sub.String()),
					zap.String("handlerName", hName),
					zap.String("pubsubMessageId", msg.ID),
				)
				msg.Nack()

				p.logger.Debug(
					"cancelling panicked subscription context",
					zap.String("subscription", sub.String()),
					zap.String("handlerName", hName),
					zap.String("pubsubMessageId", msg.ID),
				)
				// Cancel the context will remove stop the subscription from receiving messages.
				cancel()
			}
		}()

		p.logger.Debug("unmarshaling received message", zap.String("pubsubMessageId", msg.ID))
		dbzm := debeziumMessage{}
		err := json.Unmarshal(msg.Data, &dbzm)
		if err != nil {
			p.logger.Error(
				"unable to unmarshal received message: acking it, so we don't process it again",
				zap.Error(err),
				zap.String("handlerName", hName),
			)

			msg.Ack()
			return
		}

		if dbzm.Payload.Type == "" {
			p.logger.Error(
				"empty event type: acking it so, we don't process it again.",
				zap.String("pubsubMessageId", msg.ID),
				zap.String("messageId", dbzm.Payload.ID),
				zap.String("handlerName", hName),
			)

			msg.Ack()
			return
		}

		p.logger.Debug("getting event from registry", zap.String("eventType", dbzm.Payload.Type))
		// Returned event is a pointer.
		event, err := ship.GetEvent(dbzm.Payload.Type)
		if err != nil {
			p.logger.Warn(
				"event is not registered: replay the event for reprocessing, acking it for now.",
				zap.Error(err),
				zap.String("eventType", dbzm.Payload.Type),
				zap.String("handlerName", hName),
			)

			msg.Ack()
			return
		}

		p.logger.Debug("unmarshaling payload data", zap.String("eventType", dbzm.Payload.Type))
		mErr := json.Unmarshal([]byte(dbzm.Payload.Data), event)
		if mErr != nil {
			// Check whether the error is due to invalid type error. This could
			// happen if a field type does not match with event field.
			if _, ok := mErr.(*json.UnmarshalTypeError); ok {
				p.logger.Error(
					"[BUG]: invalid field type in event data:"+
						" replay the event for reprocessing, acking it for now.",
					zap.Error(mErr),
					zap.String("eventType", dbzm.Payload.Type),
					zap.String("handlerName", hName),
				)

				msg.Ack()
				return
			}

			p.logger.Error(
				"unable to unmarshal event data: acking it",
				zap.Error(mErr),
				zap.String("eventType", dbzm.Payload.Type),
				zap.String("handlerName", hName),
			)

			// Acknowleging invalid data.
			msg.Ack()
			return
		}

		p.logger.Debug(
			"sending message to the handler",
			zap.String("eventType", dbzm.Payload.Type),
			zap.String("handlerName", hName),
		)
		hErr := h.HandleMessage(ctx, &ship.Message{
			ID:            dbzm.Payload.ID,
			Metadata:      dbzm.Payload.Metadata,
			Type:          dbzm.Payload.Type,
			AggregateID:   dbzm.Payload.AggregateID,
			AggregateType: dbzm.Payload.AggregateType,
			Data:          event,
			At:            dbzm.Payload.At,
			Version:       dbzm.Payload.Version,
		})

		if hErr != nil {
			p.logger.Error("handler could not process message", zap.Error(hErr))
			msg.Nack()
			return
		}

		// Get bytes buffer from pool and reset it to empty out any garbage.
		buff := _bufferPool.Get().(*bytes.Buffer)
		buff.Reset()

		buff.WriteString("acknowledging msg with event type: ")
		buff.WriteString(dbzm.Payload.Type)

		ackMsg := buff.String()

		_bufferPool.Put(buff)

		p.logger.Debug(
			ackMsg,
			zap.String("eventType", dbzm.Payload.Type),
			zap.String("handlerName", hName),
			zap.String("pubsubMessageId", msg.ID),
			zap.String("messageId", dbzm.Payload.ID),
		)
		msg.Ack()
	})
	if err != nil {
		p.logger.Error("unable to receive messages from subscription", zap.Error(err))
		p.errCh <- err
		return
	}
}
