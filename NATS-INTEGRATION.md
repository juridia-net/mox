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
	
	# Optional: Delete emails from local mailbox after storing in NATS
	# WARNING: Use with caution! Emails will only exist in NATS.
	DeleteAfterStore: false
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
- **DeleteAfterStore**: Delete emails from local mailbox after storing in NATS (default: false)

## How It Works

### Standard Mode (DeleteAfterStore: false)
1. When an email is successfully delivered to a mailbox, mox will asynchronously store a copy in the configured NATS object store bucket
2. Each email is stored with a unique object name: `msg-{messageID}-{timestamp}`
3. The object includes metadata with the message ID and description
4. Storage happens asynchronously to avoid impacting email delivery performance
5. If NATS is unavailable, errors are logged but email delivery continues normally
6. Emails are kept both locally and in NATS

### Forward-Only Mode (DeleteAfterStore: true)
1. When an email is successfully delivered, mox will synchronously store it in NATS first
2. Only after successful NATS storage, the email is marked as expunged (deleted) from the local mailbox
3. If NATS storage fails, the email delivery fails and the email remains in the local mailbox
4. Emails exist only in NATS - no local storage
5. This mode turns mox into a NATS email forwarder

**⚠️ WARNING**: Forward-only mode should be used with caution. If NATS becomes unavailable, new emails will be rejected. Ensure NATS has proper backup and high availability.

## Object Naming Convention

Objects are stored with the following naming pattern:
```
msg-{messageID}-{unixTimestamp}
```

For example: `msg-12345-1672531200`

## Error Handling

### Standard Mode (DeleteAfterStore: false)
- If NATS is not configured, the feature is silently disabled
- If NATS connection fails during startup, an error is logged but mox continues to start
- If NATS becomes unavailable during operation, storage errors are logged but email delivery is not affected
- The system will automatically reconnect to NATS when it becomes available again

### Forward-Only Mode (DeleteAfterStore: true)
- If NATS is not configured, emails will be delivered normally to local mailboxes
- If NATS connection fails during startup, mox startup will continue but emails won't be forwarded
- If NATS becomes unavailable during operation, new email deliveries will fail with an error
- Email delivery will resume when NATS becomes available again

## Performance Considerations

### Standard Mode (DeleteAfterStore: false)
- Message storage happens asynchronously in a separate goroutine
- No impact on email delivery performance
- Network timeouts prevent NATS issues from blocking email operations
- Automatic reconnection with exponential backoff

### Forward-Only Mode (DeleteAfterStore: true)
- Message storage happens synchronously during email delivery
- Email delivery time includes NATS storage time
- Faster local disk usage (messages are deleted after forwarding)
- Requires reliable NATS connection for email delivery

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
- **Info**: "message forwarded to NATS and deleted locally" - Forward-only mode success
- **Error**: "failed to store message in NATS before deletion" - Forward-only mode NATS failure
- **Error**: "failed to delete message after NATS storage" - Forward-only mode deletion failure

## Catchall Address Integration

Mox supports catchall addresses that will also be stored in NATS when configured. This is useful for capturing emails sent to non-existent addresses within your domain.

### Types of Catchall Addresses

#### 1. Domain-wide Catchall
Catches all emails to non-existent addresses in a domain:

**domains.conf:**
```
Accounts:
  catchall-account:
    Domain: example.com
    Destinations:
      # This catches ALL non-existent addresses @example.com
      @example.com: nil
      # You can also have specific addresses
      admin@example.com: nil
```

#### 2. Localpart Catchall Separators  
Allows addresses with separators to be delivered to the base address:

**domains.conf:**
```
Domains:
  example.com:
    LocalpartCatchallSeparator: +
    # Now user+anything@example.com delivers to user@example.com
    
Accounts:
  user-account:
    Domain: example.com
    Destinations:
      user@example.com: nil
```

### NATS Storage Behavior

- **Domain-wide catchall**: Emails to `unknown@example.com` → Delivered to catchall account → **Stored in NATS**
- **Localpart catchall**: Emails to `user+tag@example.com` → Delivered to `user@example.com` → **Stored in NATS**
- **Forward-only mode**: Works with catchall addresses - they will be forwarded to NATS and deleted locally

### Example Configuration

Complete example with NATS integration and catchall:

**mox.conf:**
```
NATS: {
	URL: nats://localhost:4222
	BucketName: company-emails
	DeleteAfterStore: false  # Keep local copies
}
```

**domains.conf:**
```
Domains:
  company.com:
    LocalpartCatchallSeparator: +

Accounts:
  # Regular user account
  john:
    Domain: company.com
    Destinations:
      john@company.com: nil
      
  # Catchall account for unknown addresses
  catchall:
    Domain: company.com
    Destinations:
      @company.com: nil  # Catches unknown@company.com, typos@company.com, etc.
```

With this setup:
- `john@company.com` → Delivered to john's mailbox → Stored in NATS
- `john+newsletter@company.com` → Delivered to john's mailbox → Stored in NATS  
- `unknown@company.com` → Delivered to catchall mailbox → Stored in NATS
- `typos@company.com` → Delivered to catchall mailbox → Stored in NATS

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
	
	# For testing forward-only mode (use with caution!)
	# DeleteAfterStore: true
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
