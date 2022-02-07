// Package gcp contains an implementation of ship.Publisher and ship.Subscriber
// interface.
package gcp

import (
	"context"
	"sync"

	"cloud.google.com/go/pubsub"
	"github.com/pkg/errors"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"google.golang.org/grpc"
)

// Option is an option setter used to configure creation.
type Option func(*PubSub) error

// WithCreateTopic toggle topic creation if it does not exists.
func WithCreateTopic(create bool) Option {
	return func(p *PubSub) error {
		p.createTopic = create
		return nil
	}
}

// WithCreateSubscription toggle subscription creation if it does not exists.
func WithCreateSubscription(create bool) Option {
	return func(p *PubSub) error {
		p.createSub = create
		return nil
	}
}

// WithEndpoint changes the pubsub endpoint.
func WithEndpoint(endpoint string) Option {
	return func(p *PubSub) error {
		p.endpoint = endpoint
		return nil
	}
}

// WithLogger attaches a zap logger.
func WithLogger(logger *zap.Logger) Option {
	return func(p *PubSub) error {
		p.logger = logger.Named("gcp-pubsub")
		return nil
	}
}

// WithGRPCConn uses the provided connection instead of the default one.
// This function is useful for connecting with  mock server.
func WithGRPCConn(conn *grpc.ClientConn) Option {
	return func(p *PubSub) error {
		p.conn = conn
		return nil
	}
}

// PubSub is a wrapper over GCP PubSub.
type PubSub struct {
	projectID   string
	endpoint    string
	client      *pubsub.Client
	ctx         context.Context
	cancelFn    context.CancelFunc
	topics      map[string]*pubsub.Topic
	topicsMu    sync.RWMutex
	createTopic bool
	createSub   bool
	logger      *zap.Logger
	wg          sync.WaitGroup
	errCh       chan error
	conn        *grpc.ClientConn
}

const errorBufferLimit = 10

// NewClient creates an instance of GCP PubSub.
// All methods are thread-safe until mentioned specifically.
func NewClient(
	projectID string, options ...Option,
) (*PubSub, error) {
	p := &PubSub{
		projectID: projectID,
		logger:    zap.NewNop(),
		topics:    make(map[string]*pubsub.Topic),
		errCh:     make(chan error, errorBufferLimit),
	}

	// Apply configuration options.
	for _, opt := range options {
		if opt == nil {
			continue
		}
		if err := opt(p); err != nil {
			return nil, errors.Wrap(err, "could not apply option")
		}
	}

	// Create a cancelable context.
	ctx, cancel := context.WithCancel(context.Background())
	p.ctx = ctx
	p.cancelFn = cancel

	pubsubOpts := []option.ClientOption{}
	if p.endpoint != "" {
		p.logger.Info("changing the pubsub endpoint", zap.String("endpoint", p.endpoint))
		pubsubOpts = append(pubsubOpts, option.WithEndpoint(p.endpoint))
	}

	if p.conn != nil {
		p.logger.Info("changing the pubsub gRPC connection")
		pubsubOpts = append(pubsubOpts, option.WithGRPCConn(p.conn))
	}

	// Create the GCP pubsub client.
	p.logger.Info("creating a pubsub client", zap.String("project", projectID))
	client, err := pubsub.NewClient(ctx, projectID, pubsubOpts...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to create pubsub client")
	}

	p.client = client

	return p, nil
}

// Stop stops the pubsub gracefully.
func (p *PubSub) Stop() error {
	p.topicsMu.Lock()
	for _, topic := range p.topics {
		if topic != nil {
			p.logger.Debug("closing topic", zap.String("topic", topic.String()))
			topic.Stop()
		}
	}
	p.topicsMu.Unlock()

	p.logger.Debug("cancelling context")
	p.cancelFn()

	p.logger.Info("waiting for subscription to finish")
	p.wg.Wait()

	return p.client.Close()
}
