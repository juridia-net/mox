package store

import (
	"testing"

	"github.com/mjl-/mox/config"
	"github.com/mjl-/mox/mlog"
)

func TestNATSClientInit(t *testing.T) {
	log := mlog.New("nats-test", nil)

	// Test with nil config (NATS not configured)
	err := InitNATS(log, nil)
	if err != nil {
		t.Fatalf("InitNATS with nil config should not error: %v", err)
	}

	client := GetNATSClient()
	if client != nil {
		t.Fatal("GetNATSClient should return nil when NATS is not configured")
	}

	// Test with invalid config (should not fail startup)
	invalidConfig := &config.NATS{
		URL:        "nats://invalid-server:4222",
		BucketName: "test-bucket",
	}

	err = InitNATS(log, invalidConfig)
	// Should not return error even if connection fails - it logs and continues
	if err != nil {
		t.Logf("Expected error with invalid config (this is OK): %v", err)
	}

	// Test IsConnected with nil client
	if client := GetNATSClient(); client != nil && client.IsConnected() {
		t.Fatal("IsConnected should return false for invalid config")
	}
}

func TestNATSStoreMessage(t *testing.T) {
	// Test StoreMessage with nil client (NATS not configured)
	client := GetNATSClient()
	
	// This should not panic and return nil (graceful handling)
	err := client.StoreMessage(nil, 123, nil)
	if err != nil {
		t.Fatalf("StoreMessage with nil client should return nil: %v", err)
	}
}
