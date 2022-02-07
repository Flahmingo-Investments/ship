package gcp

import (
	"context"
	"os"
	"testing"

	"cloud.google.com/go/pubsub"
	"github.com/Flahmingo-Investments/ship"
	"github.com/stretchr/testify/assert"
)

func TestPubSub_Subscribe(t *testing.T) {
	testCases := []struct {
		name         string
		fn           func(ctx context.Context, m *ship.Message) error
		topic        string
		subscription string
		configure    func(t *testing.T, s *suite)
		checkResults func(t *testing.T, err error)
	}{
		{
			name: "should return error: subscription some-subscription does not exists",
			fn: func(ctx context.Context, m *ship.Message) error {
				return nil
			},
			topic:        "some-topic",
			subscription: "some-subscription",
			configure: func(t *testing.T, s *suite) {
			},
			checkResults: func(t *testing.T, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "subscription some-subscription does not exists")
			},
		},
		{
			name: "should attach a message handler to a subscription",
			fn: func(ctx context.Context, m *ship.Message) error {
				return nil
			},
			topic:        "some-topic",
			subscription: "some-subscription",
			configure: func(t *testing.T, s *suite) {
				topic, err := s.client.client.CreateTopic(context.Background(), "some-topic")
				assert.NoError(t, err)

				_, err = s.client.client.CreateSubscription(
					context.Background(),
					"some-subscription",
					pubsub.SubscriptionConfig{
						Topic: topic,
					},
				)
				assert.NoError(t, err)
			},
			checkResults: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			suite := newTestSuite(t)
			defer suite.Teardown(t)

			tc.configure(t, suite)

			err := suite.client.Subscribe("some-subscription", ship.MessageHandlerFunc(tc.fn))
			tc.checkResults(t, err)
		})
	}
}

func TestPubSub_SubscribeHandler(t *testing.T) {
	testCases := []struct {
		name         string
		fn           func(ctx context.Context, m *ship.Message) error
		topic        string
		subscription string
		configure    func(t *testing.T, s *suite)
		checkResults func(t *testing.T, err error)
	}{
		{
			name: "should recover from panic",
			fn: func(ctx context.Context, m *ship.Message) error {
				t.Logf("\n\ncalled\n\n")
				panic("this function would panic")
			},
			topic:        "some-topic",
			subscription: "some-subscription",
			configure: func(t *testing.T, s *suite) {
				ctx := context.Background()

				topic, err := s.client.client.CreateTopic(ctx, "some-topic")
				assert.NoError(t, err)

				_, err = s.client.client.CreateSubscription(
					ctx,
					"some-subscription",
					pubsub.SubscriptionConfig{
						Topic: topic,
					},
				)
				assert.NoError(t, err)

				data, err := os.ReadFile("testdata/valid_data.fixture")
				assert.NoError(t, err)

				res := topic.Publish(ctx, &pubsub.Message{
					Data: data,
				})
				assert.NotNil(t, res)

				id, err := res.Get(ctx)
				assert.NoError(t, err)
				assert.NotEmpty(t, id)
			},
			checkResults: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "should ack message on invalid data",
			fn: func(ctx context.Context, m *ship.Message) error {
				return nil
			},
			topic:        "some-topic",
			subscription: "some-subscription",
			configure: func(t *testing.T, s *suite) {
				ctx := context.Background()

				topic, err := s.client.client.CreateTopic(ctx, "some-topic")
				assert.NoError(t, err)

				_, err = s.client.client.CreateSubscription(
					ctx,
					"some-subscription",
					pubsub.SubscriptionConfig{
						Topic: topic,
					},
				)
				assert.NoError(t, err)

				data, err := os.ReadFile("testdata/invalid_data.fixture")
				assert.NoError(t, err)

				res := topic.Publish(ctx, &pubsub.Message{
					Data: data,
				})
				assert.NotNil(t, res)

				id, err := res.Get(ctx)
				assert.NoError(t, err)
				assert.NotEmpty(t, id)
			},
			checkResults: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
		{
			name: "should recover from panic",
			fn: func(ctx context.Context, m *ship.Message) error {
				t.Logf("\n\ncalled\n\n")
				panic("this function would panic")
			},
			topic:        "some-topic",
			subscription: "some-subscription",
			configure: func(t *testing.T, s *suite) {
				ctx := context.Background()

				topic, err := s.client.client.CreateTopic(ctx, "some-topic")
				assert.NoError(t, err)

				_, err = s.client.client.CreateSubscription(
					ctx,
					"some-subscription",
					pubsub.SubscriptionConfig{
						Topic: topic,
					},
				)
				assert.NoError(t, err)

				data, err := os.ReadFile("testdata/valid_data.fixture")
				assert.NoError(t, err)

				res := topic.Publish(ctx, &pubsub.Message{
					Data: data,
				})
				assert.NotNil(t, res)

				id, err := res.Get(ctx)
				assert.NoError(t, err)
				assert.NotEmpty(t, id)
			},
			checkResults: func(t *testing.T, err error) {
				assert.NoError(t, err)
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			suite := newTestSuite(t)
			defer suite.Teardown(t)

			tc.configure(t, suite)

			err := suite.client.Subscribe("some-subscription", ship.MessageHandlerFunc(tc.fn))
			tc.checkResults(t, err)
		})
	}
}
