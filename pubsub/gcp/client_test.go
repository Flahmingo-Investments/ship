// Package gcp contains an implementation of ship.Publisher and ship.Subscriber
// interface.
package gcp

import (
	"context"
	"errors"
	"testing"
	"time"

	"cloud.google.com/go/pubsub/pstest"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func TestNewClient(t *testing.T) {
	testCases := []struct {
		name          string
		configureOpts func(*testing.T, *pstest.Server) []Option
		checkReturns  func(*testing.T, *PubSub, error)
	}{
		{
			name: "should return error: could not apply option",
			checkReturns: func(t *testing.T, ps *PubSub, err error) {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "could not apply option")

				assert.Nil(t, ps)
			},
			configureOpts: func(t *testing.T, s *pstest.Server) []Option {
				return []Option{
					func(ps *PubSub) error {
						return errors.New("some error")
					},
				}
			},
		},
		{
			name: "should return a new client",
			checkReturns: func(t *testing.T, ps *PubSub, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, ps)
			},
			configureOpts: func(t *testing.T, s *pstest.Server) []Option {
				conn, err := grpc.Dial(s.Addr, grpc.WithInsecure())
				assert.NoError(t, err)

				return []Option{
					WithGRPCConn(conn),
					WithEndpoint("some endpoint"),
					WithCreateSubscription(true),
					WithCreateTopic(true),
					nil,
					WithLogger(zap.NewNop()),
				}
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			server := pstest.NewServer()
			defer server.Close()

			opts := tc.configureOpts(t, server)

			client, err := NewClient("some-id", opts...)
			tc.checkReturns(t, client, err)
		})
	}
}

func TestPubSub_Stop(t *testing.T) {
	testCases := []struct {
		name          string
		configureOpts func(*testing.T, *pstest.Server) []Option
		check         func(*testing.T, *PubSub, error)
	}{
		{
			name: "should return no error on stop",
			check: func(t *testing.T, ps *PubSub, err error) {
				assert.NoError(t, err)
				assert.NotNil(t, ps)
			},
			configureOpts: func(t *testing.T, s *pstest.Server) []Option {
				conn, err := grpc.Dial(s.Addr, grpc.WithInsecure())
				assert.NoError(t, err)

				return []Option{
					WithGRPCConn(conn),
				}
			},
		},
	}

	for i := range testCases {
		tc := testCases[i]

		t.Run(tc.name, func(t *testing.T) {
			server := pstest.NewServer()
			defer server.Close()

			opts := tc.configureOpts(t, server)

			client, err := NewClient("some-id", opts...)
			assert.NoError(t, err)

			ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
			defer timeout()

			done := make(chan struct{})

			go func(t *testing.T) {
				err := client.Stop()
				tc.check(t, client, err)

				close(done)
			}(t)

			select {
			case <-ctx.Done():
				assert.Fail(t, "unable to stop client properly")
			case <-done:
			}
		})
	}
}

type suite struct {
	internalPubSub *pstest.Server
	client         *PubSub
}

// Teardown teardowns the test suite.
func (s *suite) Teardown(t *testing.T) {
	ctx, timeout := context.WithTimeout(context.Background(), 10*time.Second)
	defer timeout()

	done := make(chan struct{})

	go func(t *testing.T) {
		err := s.client.Stop()
		assert.NoError(t, err)

		err = s.internalPubSub.Close()
		assert.NoError(t, err)

		close(done)
	}(t)

	select {
	case <-ctx.Done():
		assert.Fail(t, "unable to stop client properly")
	case <-done:
	}
}

// newTestSuite returns a test suite for easier testing.
func newTestSuite(t *testing.T) *suite {
	t.Helper()

	server := pstest.NewServer()
	conn, err := grpc.Dial(server.Addr, grpc.WithInsecure())
	assert.NoError(t, err)

	client, err := NewClient("some-id", WithGRPCConn(conn))
	assert.NoError(t, err)

	return &suite{
		internalPubSub: server,
		client:         client,
	}
}
