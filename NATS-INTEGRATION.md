# NATS Object Store Integration

This feature allows mox to automatically store a copy of every incoming email in a NATS JetStream Object Store bucket. This can be useful for:

- Centralized email archiving across multiple mox instances
- Backup and disaster recovery
- Compliance and auditing requirements
- Integration with other systems that need access to email data

## Configuration

Add the following to your `mox.conf` file:

```
NATS: {
	URL: nats://your-nats-server:4222
	BucketName: mox-emails
	
	# Optional authentication (choose one method)
	Username: your-username
	Password: your-password
	# OR
	Token: your-token
	# OR  
	CredentialsFile: /path/to/nats.creds
	
	# Optional timeouts (defaults shown)
	ConnectTimeout: 30s
	RequestTimeout: 30s
}
```

### Configuration Options

- **URL**: NATS server connection URL (required)
- **BucketName**: Object store bucket name where emails will be stored (required)
- **Username/Password**: Basic authentication credentials (optional)
- **Token**: Token-based authentication (optional)
- **CredentialsFile**: Path to NATS credentials file for JWT authentication (optional)
- **ConnectTimeout**: Timeout for initial connection (default: 30s)
- **RequestTimeout**: Timeout for object store operations (default: 30s)

## How It Works

1. When an email is successfully delivered to a mailbox, mox will asynchronously store a copy in the configured NATS object store bucket
2. Each email is stored with a unique object name: `msg-{messageID}-{timestamp}`
3. The object includes metadata with the message ID and description
4. Storage happens asynchronously to avoid impacting email delivery performance
5. If NATS is unavailable, errors are logged but email delivery continues normally

## Object Naming Convention

Objects are stored with the following naming pattern:
```
msg-{messageID}-{unixTimestamp}
```

For example: `msg-12345-1672531200`

## Error Handling

- If NATS is not configured, the feature is silently disabled
- If NATS connection fails during startup, an error is logged but mox continues to start
- If NATS becomes unavailable during operation, storage errors are logged but email delivery is not affected
- The system will automatically reconnect to NATS when it becomes available again

## Performance Considerations

- Message storage happens asynchronously in a separate goroutine
- No impact on email delivery performance
- Network timeouts prevent NATS issues from blocking email operations
- Automatic reconnection with exponential backoff

## Security

- Supports all NATS authentication methods (username/password, tokens, JWT credentials)
- Uses secure TLS connections when configured in NATS server
- No sensitive data is logged (credentials are not included in debug output)

## Monitoring

The following log messages indicate NATS status:

- **Info**: "NATS client initialized" - Successful connection and bucket setup
- **Info**: "NATS reconnected" - Automatic reconnection after network issues  
- **Error**: "NATS disconnected" - Connection lost (will attempt to reconnect)
- **Debug**: "message stored in NATS" - Individual message storage events
- **Error**: "storing message in NATS object store" - Storage failures

## Example NATS Server Setup

For testing, you can run NATS with JetStream enabled:

```bash
# Install NATS server
go install github.com/nats-io/nats-server/v2@latest

# Run with JetStream enabled
nats-server -js -sd /tmp/nats-storage
```

Then configure mox with:
```
NATS: {
	URL: nats://localhost:4222
	BucketName: mox-test-emails
}
```

## Retrieving Stored Emails

You can use the NATS CLI or any NATS client to retrieve stored emails:

```bash
# List objects in bucket
nats obj ls mox-emails

# Get a specific email
nats obj get mox-emails msg-12345-1672531200

# Watch for new emails in real-time
nats obj watch mox-emails
```
