package store

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"

	"github.com/mjl-/mox/config"
	"github.com/mjl-/mox/mlog"
)

// NATSClient manages the connection to NATS and object store operations
type NATSClient struct {
	conn   *nats.Conn
	js     jetstream.JetStream
	os     jetstream.ObjectStore
	config *config.NATS
	mu     sync.Mutex
	log    mlog.Log
}

// Config returns the NATS configuration
func (nc *NATSClient) Config() *config.NATS {
	if nc == nil {
		return nil
	}
	return nc.config
}

var (
	globalNATSClient *NATSClient
	natsOnce         sync.Once
)

// InitNATS initializes the global NATS client if NATS is configured
func InitNATS(log mlog.Log, cfg *config.NATS) error {
	if cfg == nil {
		log.Debug("NATS not configured, skipping initialization")
		return nil
	}

	var initErr error
	natsOnce.Do(func() {
		globalNATSClient, initErr = newNATSClient(log, cfg)
	})

	return initErr
}

// GetNATSClient returns the global NATS client, or nil if not configured
func GetNATSClient() *NATSClient {
	return globalNATSClient
}

// newNATSClient creates a new NATS client with the given configuration
func newNATSClient(log mlog.Log, cfg *config.NATS) (*NATSClient, error) {
	client := &NATSClient{
		config: cfg,
		log:    log,
	}

	// Set default timeouts
	connectTimeout := cfg.ConnectTimeout
	if connectTimeout == 0 {
		connectTimeout = 30 * time.Second
	}

	requestTimeout := cfg.RequestTimeout
	if requestTimeout == 0 {
		requestTimeout = 30 * time.Second
	}

	// Build connection options
	opts := []nats.Option{
		nats.Name("mox-email-server"),
		nats.Timeout(connectTimeout),
		nats.ReconnectWait(2 * time.Second),
		nats.MaxReconnects(-1), // unlimited reconnects
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				log.Errorx("NATS disconnected", err)
			} else {
				log.Info("NATS disconnected")
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			log.Info("NATS reconnected", slog.String("url", nc.ConnectedUrl()))
		}),
		nats.ClosedHandler(func(nc *nats.Conn) {
			log.Info("NATS connection closed")
		}),
	}

	// Add authentication options
	if cfg.CredentialsFile != "" {
		opts = append(opts, nats.UserCredentials(cfg.CredentialsFile))
	} else if cfg.Token != "" {
		opts = append(opts, nats.Token(cfg.Token))
	} else if cfg.Username != "" {
		opts = append(opts, nats.UserInfo(cfg.Username, cfg.Password))
	}

	// Connect to NATS
	conn, err := nats.Connect(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("connecting to NATS: %w", err)
	}
	client.conn = conn

	// Create JetStream context
	js, err := jetstream.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("creating JetStream context: %w", err)
	}
	client.js = js

	// Create or get object store
	ctx, cancel := context.WithTimeout(context.Background(), requestTimeout)
	defer cancel()

	os, err := js.ObjectStore(ctx, cfg.BucketName)
	if err != nil {
		// Try to create the bucket if it doesn't exist
		if err == jetstream.ErrBucketNotFound {
			log.Info("creating NATS object store bucket", slog.String("bucket", cfg.BucketName))
			os, err = js.CreateObjectStore(ctx, jetstream.ObjectStoreConfig{
				Bucket:      cfg.BucketName,
				Description: "Email message storage for mox mail server",
			})
		}
		if err != nil {
			conn.Close()
			return nil, fmt.Errorf("creating/accessing object store bucket %q: %w", cfg.BucketName, err)
		}
	}
	client.os = os

	log.Info("NATS client initialized", 
		slog.String("url", cfg.URL),
		slog.String("bucket", cfg.BucketName))

	return client, nil
}

// StoreMessage stores a message in the NATS object store
func (nc *NATSClient) StoreMessage(ctx context.Context, messageID int64, msgFile *os.File) error {
	if nc == nil {
		return nil // NATS not configured
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	// Generate object name using message ID and timestamp
	objectName := fmt.Sprintf("msg-%d-%d", messageID, time.Now().Unix())

	// Seek to beginning of file
	if _, err := msgFile.Seek(0, 0); err != nil {
		return fmt.Errorf("seeking to start of message file: %w", err)
	}

	// Create object metadata
	meta := jetstream.ObjectMeta{
		Name:        objectName,
		Description: fmt.Sprintf("Email message ID %d", messageID),
	}

	// Store the message in object store
	info, err := nc.os.Put(ctx, meta, msgFile)
	if err != nil {
		return fmt.Errorf("storing message in NATS object store: %w", err)
	}

	nc.log.Debug("message stored in NATS", 
		slog.String("object_name", objectName),
		slog.Int64("message_id", messageID),
		slog.Uint64("size", info.Size),
		slog.String("bucket", info.Bucket))

	return nil
}

// Close closes the NATS connection
func (nc *NATSClient) Close() error {
	if nc == nil || nc.conn == nil {
		return nil
	}

	nc.mu.Lock()
	defer nc.mu.Unlock()

	nc.conn.Close()
	return nil
}

// IsConnected returns true if the NATS client is connected
func (nc *NATSClient) IsConnected() bool {
	if nc == nil || nc.conn == nil {
		return false
	}
	return nc.conn.IsConnected()
}
