package dynamock

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/nisimpson/dynamap"
)

// TableManager manages DynamoDB tables for testing, providing automatic cleanup.
type TableManager struct {
	client *dynamodb.Client
	tables []string // track created tables for cleanup
}

// NewTableManager creates a new table manager with the given DynamoDB client.
func NewTableManager(client *dynamodb.Client) *TableManager {
	return &TableManager{
		client: client,
		tables: make([]string, 0),
	}
}

// CreateTestTable creates a table with the dynamap schema and tracks it for cleanup.
func (tm *TableManager) CreateTestTable(ctx context.Context, tableName string) error {
	local := &LocalDynamoDB{Client: tm.client}

	err := local.CreateDynamapTable(ctx, tableName)
	if err != nil {
		return err
	}

	// Track the table for cleanup
	tm.tables = append(tm.tables, tableName)
	return nil
}

// Cleanup deletes all tables created by this manager.
func (tm *TableManager) Cleanup(ctx context.Context) error {
	local := &LocalDynamoDB{Client: tm.client}

	for _, tableName := range tm.tables {
		if err := local.DeleteTable(ctx, tableName); err != nil {
			return fmt.Errorf("failed to delete table %s: %w", tableName, err)
		}
	}

	tm.tables = tm.tables[:0] // Clear the slice
	return nil
}

// GetTableNames returns the names of all tables managed by this manager.
func (tm *TableManager) GetTableNames() []string {
	names := make([]string, len(tm.tables))
	copy(names, tm.tables)
	return names
}

// WithIsolatedTable runs a test function with an isolated table that is automatically cleaned up.
// The table name is generated to be unique for the test.
func WithIsolatedTable(t *testing.T, client *dynamodb.Client, fn func(tableName string)) {
	ctx := context.Background()
	tableName := fmt.Sprintf("test-%s-%d", t.Name(), time.Now().UnixNano())

	// Create table manager for cleanup
	tm := NewTableManager(client)

	// Ensure cleanup happens even if test panics
	defer func() {
		if err := tm.Cleanup(ctx); err != nil {
			t.Errorf("Failed to cleanup table %s: %v", tableName, err)
		}
	}()

	// Create the test table
	err := tm.CreateTestTable(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to create test table %s: %v", tableName, err)
	}

	// Run the test function
	fn(tableName)
}

// WithLocalDynamoDB runs a test function with a local DynamoDB instance.
// It checks if DynamoDB Local is available and skips the test if not.
func WithLocalDynamoDB(t *testing.T, port int, fn func(local *LocalDynamoDB)) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	local := NewLocalDynamoDB(port)
	ctx := context.Background()

	// Check if DynamoDB Local is available
	if !local.IsAvailable(ctx) {
		t.Skipf("DynamoDB Local not available on port %d", port)
	}

	// Run the test function
	fn(local)
}

// WithDefaultLocalDynamoDB runs a test function with the default local DynamoDB instance (port 8000).
func WithDefaultLocalDynamoDB(t *testing.T, fn func(local *LocalDynamoDB)) {
	WithLocalDynamoDB(t, DefaultLocalPort, fn)
}

// NewTestTable generates a unique table name for testing.
func NewTestTable(prefix string) string {
	return fmt.Sprintf("%s-%d", prefix, time.Now().UnixNano())
}

// SeedTestData is a helper for seeding test data into a table.
type SeedTestData struct {
	client    *dynamodb.Client
	tableName string
}

// NewSeedTestData creates a new test data seeder.
func NewSeedTestData(client *dynamodb.Client, tableName string) *SeedTestData {
	return &SeedTestData{
		client:    client,
		tableName: tableName,
	}
}

// SeedEntity seeds a single entity into the table.
func (s *SeedTestData) SeedEntity(ctx context.Context, entity dynamap.Marshaler) error {
	table := dynamap.NewTable(s.tableName)

	putInput, err := table.MarshalPut(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal entity: %w", err)
	}

	_, err = s.client.PutItem(ctx, putInput)
	if err != nil {
		return fmt.Errorf("failed to put entity: %w", err)
	}

	return nil
}

// SeedEntityWithRefs seeds an entity and all its relationships into the table.
func (s *SeedTestData) SeedEntityWithRefs(ctx context.Context, entity dynamap.RefMarshaler) error {
	table := dynamap.NewTable(s.tableName)

	batches, err := table.MarshalBatch(entity)
	if err != nil {
		return fmt.Errorf("failed to marshal entity with refs: %w", err)
	}

	for _, batch := range batches {
		_, err = s.client.BatchWriteItem(ctx, batch)
		if err != nil {
			return fmt.Errorf("failed to batch write: %w", err)
		}
	}

	return nil
}

// SeedEntities seeds multiple entities into the table.
func (s *SeedTestData) SeedEntities(ctx context.Context, entities ...dynamap.Marshaler) error {
	for _, entity := range entities {
		if err := s.SeedEntity(ctx, entity); err != nil {
			return err
		}
	}
	return nil
}

// IntegrationTestConfig holds configuration for integration tests.
type IntegrationTestConfig struct {
	Port             int
	SkipIfNotRunning bool
	TablePrefix      string
	CleanupTimeout   time.Duration
}

// DefaultIntegrationTestConfig returns a default configuration for integration tests.
func DefaultIntegrationTestConfig() *IntegrationTestConfig {
	return &IntegrationTestConfig{
		Port:             DefaultLocalPort,
		SkipIfNotRunning: true,
		TablePrefix:      "integration-test",
		CleanupTimeout:   30 * time.Second,
	}
}

// RunIntegrationTest runs an integration test with the given configuration.
func RunIntegrationTest(t *testing.T, config *IntegrationTestConfig, fn func(local *LocalDynamoDB, tableName string)) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	if config == nil {
		config = DefaultIntegrationTestConfig()
	}

	local := NewLocalDynamoDB(config.Port)
	ctx := context.Background()

	// Check if DynamoDB Local is available
	if !local.IsAvailable(ctx) {
		if config.SkipIfNotRunning {
			t.Skipf("DynamoDB Local not available on port %d", config.Port)
		} else {
			t.Fatalf("DynamoDB Local not available on port %d", config.Port)
		}
	}

	// Generate unique table name
	tableName := NewTestTable(config.TablePrefix)

	// Create the test table
	err := local.CreateDynamapTable(ctx, tableName)
	if err != nil {
		t.Fatalf("Failed to create test table %s: %v", tableName, err)
	}

	// Ensure cleanup happens
	defer func() {
		cleanupCtx, cancel := context.WithTimeout(context.Background(), config.CleanupTimeout)
		defer cancel()

		if err := local.DeleteTable(cleanupCtx, tableName); err != nil {
			t.Errorf("Failed to cleanup table %s: %v", tableName, err)
		}
	}()

	// Run the test function
	fn(local, tableName)
}

// AssertTableExists verifies that a table exists.
func AssertTableExists(t *testing.T, client *dynamodb.Client, tableName string) {
	ctx := context.Background()

	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: &tableName,
	})

	if err != nil {
		t.Errorf("Table %s does not exist: %v", tableName, err)
	}
}

// AssertTableNotExists verifies that a table does not exist.
func AssertTableNotExists(t *testing.T, client *dynamodb.Client, tableName string) {
	ctx := context.Background()

	_, err := client.DescribeTable(ctx, &dynamodb.DescribeTableInput{
		TableName: &tableName,
	})

	if err == nil {
		t.Errorf("Table %s should not exist but it does", tableName)
	}
}
