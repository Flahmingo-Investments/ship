package gcp

import (
	"encoding/json"
	"fmt"

	"cloud.google.com/go/pubsub"
	"github.com/Flahmingo-Investments/ship"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// EnsureTopics checks whether a topic exists or not.
//
// If createTopic is `true` it will create the topic.
func (p *PubSub) EnsureTopics(topics ...string) error {
	for _, topic := range topics {
		if topic == "" {
			continue
		}

		createdTopic, err := p.topicInit(topic)
		if err != nil {
			return errors.WithStack(err)
		}

		p.cacheTopic(topic, createdTopic)
	}
	return nil
}

// topicInit initialize a topic and create it, if required.
func (p *PubSub) topicInit(topic string) (*pubsub.Topic, error) {
	t := p.client.Topic(topic)
	t.EnableMessageOrdering = true

	p.logger.Info(
		fmt.Sprintf("checking if topic (%s) exists", topic),
		zap.String("topic", topic),
	)
	exists, err := t.Exists(p.ctx)
	if err != nil {
		return nil, errors.Wrapf(
			err, "could not check if topic '%s' exists", topic,
		)
	}

	if !exists {
		if !p.createTopic {
			return nil, errors.Errorf("topic %s does not exists", topic)
		}

		p.logger.Info(
			fmt.Sprintf("creating topic %s", topic),
			zap.String("topic", topic),
		)

		createdTopic, err := p.client.CreateTopic(p.ctx, topic)
		if err != nil {
			return nil, errors.Wrapf(err, "unable to create %s topic", topic)
		}

		t = createdTopic
		t.EnableMessageOrdering = true
	}

	return t, nil
}

// cacheTopic caches a given topic.
func (p *PubSub) cacheTopic(name string, t *pubsub.Topic) {
	p.topicsMu.Lock()
	p.topics[name] = t
	p.topicsMu.Unlock()
}

// Publish publishes the message to a given topic.
func (p *PubSub) Publish(topic string, message *ship.Message) error {
	p.topicsMu.RLock()
	t, ok := p.topics[topic]
	p.topicsMu.RUnlock()

	// if topic is not cached then cache it.
	//
	// Why?
	//
	// Excerpt from pubsub.Topic documentation.
	// Avoid creating many Topic instances if you use them to publish.
	if !ok {
		p.logger.Debug(
			fmt.Sprintf("topic (%s) is not cached, caching it", topic),
			zap.String("topic", topic),
		)
		t = p.client.Topic(topic)
		t.EnableMessageOrdering = true

		p.cacheTopic(topic, t)
	}

	p.logger.Debug(
		"publishing message to topic", zap.String("topic", topic),
	)

	data, err := json.Marshal(message.Data)
	if err != nil {
		return errors.Wrap(err, "unable to marshal message to bytes")
	}

	pbMsg := pubsub.Message{
		Data:        data,
		Attributes:  message.Metadata,
		OrderingKey: "",
	}
	res := t.Publish(p.ctx, &pbMsg)

	p.logger.Debug(
		"checking if message was published successfully",
		zap.String("topic", topic),
	)
	if id, err := res.Get(p.ctx); err != nil {
		p.logger.Error(
			"unable to publish message",
			zap.String("topic", topic),
			zap.String("messageId", id),
		)
		return errors.Wrap(err, "could not publish message")
	}

	return nil
}
