package mongo

import (
	"context"
	"fmt"
	"time"

	"github.com/0dev1337/tmpvault-backend/pkg/logging"
	driver "go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Client owns the lifecycle of a MongoDB connection.
type Client struct {
	cli *driver.Client
}

// Connect dials MongoDB, verifies with ping, and returns a managed client.
func Connect(ctx context.Context, uri string) (*Client, error) {
	if uri == "" {
		return nil, fmt.Errorf("mongo: empty URI")
	}

	clientOpts := options.Client().ApplyURI(uri)
	cli, err := driver.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo connect: %w", err)
	}

	pingCtx := ctx
	if _, hasDeadline := ctx.Deadline(); !hasDeadline {
		var cancel context.CancelFunc
		pingCtx, cancel = context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
	}

	if err := cli.Ping(pingCtx, nil); err != nil {
		_ = cli.Disconnect(context.Background())
		return nil, fmt.Errorf("mongo ping: %w", err)
	}

	logging.Logger.Info().Msg("Connected to MongoDB")
	return &Client{cli: cli}, nil
}

// Driver exposes the underlying mongo driver client for advanced use.
func (c *Client) Driver() *driver.Client {
	if c == nil {
		return nil
	}
	return c.cli
}

// Disconnect closes the client.
func (c *Client) Disconnect(ctx context.Context) error {
	if c == nil || c.cli == nil {
		return nil
	}
	if err := c.cli.Disconnect(ctx); err != nil {
		return fmt.Errorf("mongo disconnect: %w", err)
	}
	logging.Logger.Info().Msg("Disconnected from MongoDB")
	return nil
}

// DB returns a handle for the named database.
func (c *Client) DB(name string) *Database {
	if c == nil || c.cli == nil {
		return nil
	}
	return &Database{db: c.cli.Database(name)}
}


