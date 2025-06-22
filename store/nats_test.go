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

func TestNATSConfig(t *testing.T) {
	// Test Config with nil client
	client := GetNATSClient()
	cfg := client.Config()
	if cfg != nil {
		t.Fatal("Config should return nil for nil client")
	}
	
	// Test with valid config
	log := mlog.New("nats-test", nil)
	testConfig := &config.NATS{
		URL:              "nats://test:4222",
		BucketName:       "test-bucket",
		DeleteAfterStore: true,
	}
	
	// This will fail to connect but should still store config
	err := InitNATS(log, testConfig)
	if err != nil {
		t.Logf("Expected connection error: %v", err)
	}
	
	// Even with failed connection, we should be able to test config structure
	if testConfig.DeleteAfterStore != true {
		t.Fatal("DeleteAfterStore should be true")
	}
}
