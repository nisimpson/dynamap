package dynamock

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
)

func TestNewLocalClient(t *testing.T) {
	client := NewLocalClient(8000)

	if client == nil {
		t.Fatal("NewLocalClient returned nil")
	}

	// We can't test actual connectivity without DynamoDB Local running,
	// but we can verify the client was created
}

func TestNewLocalDynamoDB(t *testing.T) {
	local := NewLocalDynamoDB(8000)

	if local == nil {
		t.Fatal("NewLocalDynamoDB returned nil")
	}

	if local.Client == nil {
		t.Error("Client is nil")
	}

	if local.Endpoint != "http://localhost:8000" {
		t.Errorf("expected endpoint http://localhost:8000, got %s", local.Endpoint)
	}

	if local.Port != 8000 {
		t.Errorf("expected port 8000, got %d", local.Port)
	}
}

func TestNewDefaultLocalClient(t *testing.T) {
	client := NewDefaultLocalClient()

	if client == nil {
		t.Fatal("NewDefaultLocalClient returned nil")
	}
}

func TestNewDefaultLocalDynamoDB(t *testing.T) {
	local := NewDefaultLocalDynamoDB()

	if local == nil {
		t.Fatal("NewDefaultLocalDynamoDB returned nil")
	}

	if local.Port != DefaultLocalPort {
		t.Errorf("expected port %d, got %d", DefaultLocalPort, local.Port)
	}
}

func TestNewLocalClientFromConfig(t *testing.T) {
	cfg := aws.Config{
		Region: "us-west-2",
	}

	client := NewLocalClientFromConfig(cfg, 8001)

	if client == nil {
		t.Fatal("NewLocalClientFromConfig returned nil")
	}
}

// TestLocalDynamoDB_Integration tests the local DynamoDB functionality.
// This test is skipped by default since it requires DynamoDB Local to be running.
func TestLocalDynamoDB_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	local := NewLocalDynamoDB(8000)
	ctx := context.Background()

	// Check if DynamoDB Local is available
	if !local.IsAvailable(ctx) {
		t.Skip("DynamoDB Local not available on port 8000")
	}

	// Test table creation
	tableName := "test-table-" + time.Now().Format("20060102150405")

	err := local.CreateDynamapTable(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to create table: %v", err)
	}

	// Verify table exists
	tables, err := local.ListTables(ctx)
	if err != nil {
		t.Fatalf("Failed to list tables: %v", err)
	}

	found := false
	for _, table := range tables {
		if table == tableName {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Table %s not found in table list", tableName)
	}

	// Clean up
	err = local.DeleteTable(ctx, tableName)
	if err != nil {
		t.Errorf("Failed to delete table: %v", err)
	}
}

// TestLocalDynamoDB_WaitForAvailable tests the availability checking.
func TestLocalDynamoDB_WaitForAvailable(t *testing.T) {
	local := NewLocalDynamoDB(9999) // Use a port that's likely not in use
	ctx := context.Background()

	// This should timeout quickly since nothing is running on port 9999
	err := local.WaitForAvailable(ctx, 1*time.Second)
	if err == nil {
		t.Error("Expected WaitForAvailable to timeout, but it succeeded")
	}
}

// TestLocalDynamoDB_IsAvailable tests the availability check.
func TestLocalDynamoDB_IsAvailable(t *testing.T) {
	local := NewLocalDynamoDB(9999) // Use a port that's likely not in use
	ctx := context.Background()

	// This should return false since nothing is running on port 9999
	if local.IsAvailable(ctx) {
		t.Error("Expected IsAvailable to return false for unused port")
	}
}

// TestMustNewLocalClient_Panic tests that MustNewLocalClient panics when it can't connect.
func TestMustNewLocalClient_Panic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected MustNewLocalClient to panic, but it didn't")
		}
	}()

	// This should panic since nothing is running on port 9999
	MustNewLocalClient(9999)
}

// Example of how to use the local client in integration tests
func ExampleNewLocalClient() {
	// Create a client for DynamoDB Local running on port 8000
	client := NewLocalClient(8000)

	// Use the client in your tests
	ctx := context.Background()
	_, err := client.ListTables(ctx, nil)
	if err != nil {
		// Handle error - DynamoDB Local might not be running
	}
}

// Example of how to use LocalDynamoDB for integration testing
func ExampleLocalDynamoDB() {
	local := NewLocalDynamoDB(8000)
	ctx := context.Background()

	// Check if DynamoDB Local is available
	if !local.IsAvailable(ctx) {
		// Start DynamoDB Local or skip the test
		return
	}

	// Create a test table
	tableName := "integration-test-table"
	err := local.CreateDynamapTable(ctx, tableName)
	if err != nil {
		// Handle error
		return
	}

	// Run your tests...

	// Clean up
	err = local.DeleteTable(ctx, tableName)
	if err != nil {
		// Handle cleanup error
	}
}
